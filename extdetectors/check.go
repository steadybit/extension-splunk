/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdetectors

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-splunk/config"
	"strings"
	"time"
)

type DetectorStateCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[DetectorCheckState]           = (*DetectorStateCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[DetectorCheckState] = (*DetectorStateCheckAction)(nil)
)

const (
	NoIncident       = "No Incidents"
	Anomalous        = "ANOMALOUS"
	ManuallyResolved = "MANUALLY_RESOLVED"
	Ok               = "OK"
	Stopped          = "STOPPED"
)

type DetectorCheckState struct {
	DetectorId            string
	DetectorName          string
	CheckNewIncidentsOnly bool
	Start                 time.Time
	End                   time.Time
	ExpectedState         string
	StateCheckMode        string
	StateCheckSuccess     bool
}

func NewDetectorStateCheckAction() action_kit_sdk.Action[DetectorCheckState] {
	return &DetectorStateCheckAction{}
}

func (m *DetectorStateCheckAction) NewEmptyState() DetectorCheckState {
	return DetectorCheckState{}
}

func (m *DetectorStateCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.check", TargetType),
		Label:       "Check Detector Incidents",
		Description: "Check if the detector have active incidents.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          TargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionAll),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find Detector by id"),
					Query:       "splunk.detector.id=\"\"",
				},
			}),
		}),
		Technology:  extutil.Ptr("Splunk"),
		Category:    extutil.Ptr("Splunk"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "checkNewIncidentsOnly",
				Label:        "Check New Incidents Only",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(false),
			},
			{
				Name:        "expectedStateList",
				Label:       "Expected Incident Anomaly State",
				Description: extutil.Ptr(""),
				Type:        action_kit_api.ActionParameterTypeString,
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "No Incidents At All",
						Value: NoIncident,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Anomalous",
						Value: Anomalous,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Manually resolved",
						Value: ManuallyResolved,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Ok",
						Value: Ok,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Stopped",
						Value: Stopped,
					},
				}),
				Required: extutil.Ptr(true),
				Order:    extutil.Ptr(2),
			},
			{
				Name:         "stateCheckMode",
				Label:        "State Check Mode",
				Description:  extutil.Ptr("How often should the state be checked ?"),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr(stateCheckModeAllTheTime),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "All the time",
						Value: stateCheckModeAllTheTime,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least once",
						Value: stateCheckModeAtLeastOnce,
					},
				}),
				Required: extutil.Ptr(true),
				Order:    extutil.Ptr(3),
			},
		},
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "Splunk Detector Incidents State",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: "splunk.metric.id",
				},
				Label: action_kit_api.StateOverTimeWidgetLabelConfig{
					From: "splunk.metric.label",
				},
				State: action_kit_api.StateOverTimeWidgetStateConfig{
					From: "state",
				},
				Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
					From: "tooltip",
				},
				Url: extutil.Ptr(action_kit_api.StateOverTimeWidgetUrlConfig{
					From: extutil.Ptr("url"),
				}),
				Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: extutil.Ptr(true),
				}),
			},
		}),
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

func (m *DetectorStateCheckAction) Prepare(_ context.Context, state *DetectorCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	DetectorId := request.Target.Attributes[attributeID]
	if len(DetectorId) == 0 {
		return nil, extutil.Ptr(extension_kit.ToError("Target is missing the '"+attributeID+"' attribute.", nil))
	}

	duration := request.Config["duration"].(float64)
	start := time.Now()
	end := start.Add(time.Millisecond * time.Duration(duration))

	var expectedState string
	if request.Config["expectedStateList"] != nil {
		expectedState = fmt.Sprintf("%v", request.Config["expectedStateList"])
	}

	var stateCheckMode string
	if request.Config["stateCheckMode"] != nil {
		stateCheckMode = fmt.Sprintf("%v", request.Config["stateCheckMode"])
	}

	if request.Config["checkNewIncidentsOnly"] != nil {
		state.CheckNewIncidentsOnly = extutil.ToBool(request.Config["checkNewIncidentsOnly"])
	}

	state.DetectorId = DetectorId[0]
	state.DetectorName = request.Target.Attributes[attributeName][0]
	state.Start = start
	state.End = end
	state.ExpectedState = expectedState
	state.StateCheckMode = stateCheckMode

	return nil, nil
}

func (m *DetectorStateCheckAction) Start(_ context.Context, _ *DetectorCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (m *DetectorStateCheckAction) Status(ctx context.Context, state *DetectorCheckState) (*action_kit_api.StatusResult, error) {
	return DetectorCheckStatus(ctx, state, RestyClient)
}

func DetectorCheckStatus(ctx context.Context, state *DetectorCheckState, client *resty.Client) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	var incidents []Incident

	uri := "/v2/detector/" + state.DetectorId + "/incidents"
	res, err := client.R().
		SetContext(ctx).
		SetResult(&incidents).
		Get(uri)

	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to retrieve detector incidents from Splunk for detector %s with uri %s. Full response: %v", state.DetectorId, uri, res.String()), err))
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("Splunk API responded with unexpected status code %d while retrieving Detector incidents for detector %s. Full response: %v", res.StatusCode(), state.DetectorId, res.String())
	} else {
		if state.CheckNewIncidentsOnly {
			var filteredIncidents []Incident
			for _, incident := range incidents {
				if time.UnixMilli(incident.AnomalyStateUpdateTimestamp).After(state.Start) {
					filteredIncidents = append(filteredIncidents, incident)
				}
			}
			incidents = filteredIncidents
		}
	}

	completed := now.After(state.End)
	var checkError *action_kit_api.ActionKitError

	if len(state.ExpectedState) > 0 {
		if state.StateCheckMode == stateCheckModeAllTheTime {
			for _, incident := range incidents {
				if state.ExpectedState != incident.AnomalyState {
					checkError = extutil.Ptr(action_kit_api.ActionKitError{
						Title: fmt.Sprintf("One of the incidents of the detector '%s' has state '%s' whereas '%s' is expected.",
							state.DetectorName,
							incident.AnomalyState,
							state.ExpectedState),
						Status: extutil.Ptr(action_kit_api.Failed),
					})
					break
				}
			}
			if state.ExpectedState != NoIncident {
				if len(incidents) == 0 {
					checkError = extutil.Ptr(action_kit_api.ActionKitError{
						Title: fmt.Sprintf("No incidents found for detector '%s' whereas incident(s) with '%s' state is expected.",
							state.DetectorName,
							state.ExpectedState),
						Status: extutil.Ptr(action_kit_api.Failed),
					})
				}
			}
		} else if state.StateCheckMode == stateCheckModeAtLeastOnce {
			for _, incident := range incidents {
				if state.ExpectedState == incident.AnomalyState {
					state.StateCheckSuccess = true
				}
			}

			if completed && !state.StateCheckSuccess {
				checkError = extutil.Ptr(action_kit_api.ActionKitError{
					Title: fmt.Sprintf("Detector '%s' incidents didn't have status '%s' at least once.",
						state.DetectorName,
						state.ExpectedState),
					Status: extutil.Ptr(action_kit_api.Failed),
				})
			}
		}
	}

	var metrics []action_kit_api.Metric
	for _, incident := range incidents {
		metrics = append(metrics, *toMetric(state.DetectorId, state.DetectorName, incident, now))
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics:   extutil.Ptr(metrics),
	}, nil
}

func toMetric(detectorID string, detectorName string, incident Incident, now time.Time) *action_kit_api.Metric {
	var tooltip string
	var state string
	var url string

	tooltip = fmt.Sprintf("Detector incident state is: %s", incident.AnomalyState)
	url = fmt.Sprintf("%s/#/detector-wizard/%s/edit", strings.TrimRight(strings.Replace(config.Config.ApiBaseUrl, "https://api", "https://app", 1), "/"), detectorID)

	if incident.AnomalyState == Ok {
		state = "success"
	} else if incident.AnomalyState == Stopped {
		state = "warn"
	} else if incident.AnomalyState == ManuallyResolved {
		state = "success"
	} else if incident.AnomalyState == Anomalous {
		state = "danger"
	} else {
		state = "info"
	}

	return extutil.Ptr(action_kit_api.Metric{
		Name: extutil.Ptr("splunk_detector_incident_state"),
		Metric: map[string]string{
			"splunk.metric.id":    detectorID,
			"splunk.metric.label": detectorName,
			"state":               state,
			"tooltip":             tooltip,
			"url":                 url,
		},
		Timestamp: now,
		Value:     0,
	})
}
