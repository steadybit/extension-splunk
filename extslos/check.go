/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extslos

import (
	"context"
	"encoding/json"
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

type SloStateCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[SloCheckState]           = (*SloStateCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[SloCheckState] = (*SloStateCheckAction)(nil)
)

const (
	breachAlertsTriggered          = "breach"
	burnRateAlertsTriggered        = "burn rate"
	errorBudgetLeftAlertsTriggered = "error budget"
)

type SloCheckState struct {
	SloID              string
	SloName            string
	CheckNewAlertsOnly bool
	Start              time.Time
	End                time.Time
	ExpectedState      string
	StateCheckMode     string
	StateCheckSuccess  bool
}

func NewSloStateCheckAction() action_kit_sdk.Action[SloCheckState] {
	return &SloStateCheckAction{}
}

func (m *SloStateCheckAction) NewEmptyState() SloCheckState {
	return SloCheckState{}
}

func (m *SloStateCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.check", TargetType),
		Label:       "Check SLO Alerts",
		Description: "Check if the slo have active alerts.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          TargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find SLO by id"),
					Query:       "splunk.slo.id=\"\"",
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
				Name:         "checkNewAlertsOnly",
				Label:        "Check New Alerts Only",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(false),
			},
			{
				Name:        "expectedStateList",
				Label:       "Expected SLO Alerts triggered",
				Description: extutil.Ptr(""),
				Type:        action_kit_api.ActionParameterTypeString,
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "breachAlertsTriggered",
						Value: breachAlertsTriggered,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Manually resolved",
						Value: burnRateAlertsTriggered,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "errorBudgetLeftAlertsTriggered",
						Value: errorBudgetLeftAlertsTriggered,
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
				Title: "Splunk SLO Active Alerts",
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
			CallInterval: extutil.Ptr("2s"),
		}),
	}
}

func (m *SloStateCheckAction) Prepare(_ context.Context, state *SloCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	SLOId := request.Target.Attributes[attributeID]
	if len(SLOId) == 0 {
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

	if request.Config["checkNewAlertsOnly"] != nil {
		state.CheckNewAlertsOnly = extutil.ToBool(request.Config["checkNewAlertsOnly"])
	}

	state.SloID = SLOId[0]
	state.SloName = request.Target.Attributes[attributeName][0]
	state.Start = start
	state.End = end
	state.ExpectedState = expectedState
	state.StateCheckMode = stateCheckMode

	return nil, nil
}

func (m *SloStateCheckAction) Start(_ context.Context, _ *SloCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (m *SloStateCheckAction) Status(ctx context.Context, state *SloCheckState) (*action_kit_api.StatusResult, error) {
	return SLOCheckStatus(ctx, state, RestyClient)
}

func SLOCheckStatus(ctx context.Context, state *SloCheckState, client *resty.Client) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	breachAlertsTriggered := strings.Contains(state.ExpectedState, breachAlertsTriggered)
	burnRateAlertsTriggered := strings.Contains(state.ExpectedState, burnRateAlertsTriggered)
	errorBudgetLeftAlertsTriggered := strings.Contains(state.ExpectedState, errorBudgetLeftAlertsTriggered)

	jsonData, err := json.Marshal(SLOSearchConfig{
		BreachAlertsTriggered:          breachAlertsTriggered,
		BurnRateAlertsTriggered:        burnRateAlertsTriggered,
		ErrorBudgetLeftAlertsTriggered: errorBudgetLeftAlertsTriggered,
		SLOIds:                         []string{state.SloID},
	})
	if err != nil {
		return nil, extension_kit.ToError("SLOCheckStatus, marshal error", err)
	}

	var slosFound Response
	res, err := client.R().
		SetContext(ctx).
		SetResult(&slosFound).
		SetBody(jsonData).
		Post("/v2/slo/search")

	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to retrieve SLOs from Splunk for ID %s. Full response: %v", state.SloID, res.String()), err))
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("Splunk API responded with unexpected status code %d while retrieving SLOs for ID %s. Full response: %v", res.StatusCode(), state.SloID, res.String())
	} else {
		if state.CheckNewAlertsOnly {
			var filteredSlos []Slo
			for _, slo := range slosFound.Results {
				if time.UnixMilli(slo.LastUpdated).After(state.Start) {
					filteredSlos = append(filteredSlos, slo)
				}
			}
			slosFound.Results = filteredSlos
		}
	}

	completed := now.After(state.End)
	var checkError *action_kit_api.ActionKitError

	if state.StateCheckMode == stateCheckModeAllTheTime {
		if slosFound.Count == 0 {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title: fmt.Sprintf("The SLO '%s' has no alerts whereas '%s' is expected.",
					state.SloName,
					state.ExpectedState),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}

	} else if state.StateCheckMode == stateCheckModeAtLeastOnce {
		if slosFound.Count > 0 {
			state.StateCheckSuccess = true
		}

		if completed && !state.StateCheckSuccess {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title: fmt.Sprintf("The SLO '%s'  didn't triggered this type of alerts '%s' at least once.",
					state.SloName,
					state.ExpectedState),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}
	}

	var metrics []action_kit_api.Metric
	for _, slo := range slosFound.Results {
		metrics = append(metrics, *toMetric(state.ExpectedState, slo, now))
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics:   extutil.Ptr(metrics),
	}, nil
}

func toMetric(expectedState string, slo Slo, now time.Time) *action_kit_api.Metric {
	var tooltip string
	var state string
	var url string

	tooltip = fmt.Sprintf("SLO %s have active %s alert", slo.Name, expectedState)
	url = fmt.Sprintf("%s/#/alerts?query=%s", strings.TrimRight(strings.Replace(config.Config.ApiBaseUrl, "https://api", "https://app", 1), "/"), slo.ID)

	state = "danger"

	return extutil.Ptr(action_kit_api.Metric{
		Name: extutil.Ptr("splunk_detector_incident_state"),
		Metric: map[string]string{
			"splunk.metric.id":    slo.ID + "-" + expectedState,
			"splunk.metric.label": expectedState + " active alerts",
			"state":               state,
			"tooltip":             tooltip,
			"url":                 url,
		},
		Timestamp: now,
		Value:     0,
	})
}
