// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/event-kit/go/event_kit_api"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"maps"
	"net/http"
	"time"
)

func RegisterEventListenerHandlers() {
	exthttp.RegisterHttpHandler("/events/experiment-started", handle(onExperiment))
	exthttp.RegisterHttpHandler("/events/experiment-completed", handle(onExperiment))
	exthttp.RegisterHttpHandler("/events/experiment-step-started", handle(onExperimentStep))
	exthttp.RegisterHttpHandler("/events/experiment-step-completed", handle(onExperimentStep))
}

const (
	category = "USER_DEFINED"
)

var RestyClient *resty.Client

type eventHandler func(event event_kit_api.EventRequestBody) (*Event, error)

func handle(handler eventHandler) func(w http.ResponseWriter, r *http.Request, body []byte) {
	return func(w http.ResponseWriter, r *http.Request, body []byte) {

		event, err := parseBodyToEventRequestBody(body)
		if err != nil {
			exthttp.WriteError(w, extension_kit.ToError("Failed to decode event request body", err))
			return
		}

		if request, err := handler(event); err == nil {
			if request != nil {
				handlePostEvent(r.Context(), RestyClient, request)
			}
		} else {
			exthttp.WriteError(w, extension_kit.ToError(err.Error(), err))
			return
		}

		exthttp.WriteBody(w, "{}")
	}
}

func onExperiment(event event_kit_api.EventRequestBody) (*Event, error) {
	tags := getEventBaseTags(event)
	maps.Copy(tags, getExecutionTags(event))

	return &Event{
		Category:   category,
		EventType:  "Steadybit_Event",
		Properties: tags,
		Timestamp:  event.EventTime.UnixMilli(),
	}, nil
}

func onExperimentStep(event event_kit_api.EventRequestBody) (*Event, error) {
	tags := getEventBaseTags(event)
	maps.Copy(tags, getExecutionTags(event))
	maps.Copy(tags, getStepTags(*event.ExperimentStepExecution))

	return &Event{
		Category:   category,
		EventType:  "Steadybit_Event",
		Properties: tags,
		Timestamp:  event.EventTime.UnixMilli(),
	}, nil
}

func getEventBaseTags(event event_kit_api.EventRequestBody) map[string]string {
	tags := make(map[string]string)
	tags["source"] = "Steadybit"
	tags["env"] = event.Environment.Name
	tags["event"] = event.EventName
	tags["event_id"] = event.Id.String()
	tags["tenant"] = event.Tenant.Name
	tags["tenant_key"] = event.Tenant.Key

	if event.Team != nil {
		tags["team_name"] = event.Team.Name
		tags["team_key"] = event.Team.Key
	}

	return tags
}

func getExecutionTags(event event_kit_api.EventRequestBody) map[string]string {
	tags := make(map[string]string)
	if event.ExperimentExecution == nil {
		return tags
	}
	tags["exec_id"] = fmt.Sprintf("%g", event.ExperimentExecution.ExecutionId)
	tags["exp_key"] = event.ExperimentExecution.ExperimentKey
	tags["exp_name"] = event.ExperimentExecution.Name

	if event.ExperimentExecution.StartedTime.IsZero() {
		tags["started_time"] = time.Now().Format(time.RFC3339)
	} else {
		tags["started_time"] = event.ExperimentExecution.StartedTime.Format(time.RFC3339)
	}

	if event.ExperimentExecution.EndedTime != nil && !(*event.ExperimentExecution.EndedTime).IsZero() {
		tags["ended_time"] = event.ExperimentExecution.EndedTime.Format(time.RFC3339)
	}

	return tags
}

func getStepTags(step event_kit_api.ExperimentStepExecution) map[string]string {
	tags := make(map[string]string)

	if step.Type == event_kit_api.Action {
		tags["step_action_id"] = *step.ActionId
	}
	if step.ActionName != nil {
		tags["step_name"] = *step.ActionName
	}
	if step.CustomLabel != nil {
		tags["step_label"] = *step.CustomLabel
	}
	tags["step_exec_id"] = fmt.Sprintf("%.0f", step.ExecutionId)
	tags["step_exp_key"] = step.ExperimentKey
	tags["step_id"] = step.Id.String()

	return tags
}

func parseBodyToEventRequestBody(body []byte) (event_kit_api.EventRequestBody, error) {
	var event event_kit_api.EventRequestBody
	err := json.Unmarshal(body, &event)
	return event, err
}

func handlePostEvent(ctx context.Context, client *resty.Client, event *Event) {
	events := make([]*Event, 0)
	events = append(events, event)

	eventBytes, err := json.Marshal(events)
	if err != nil {
		log.Err(err).Msgf("Failed to marshal event %v. Full response: %v", event, err)
		return
	}

	res, err := client.R().
		SetContext(ctx).
		SetBody(eventBytes).
		Post("/v2/event")

	if err != nil {
		log.Err(err).Msgf("Failed to post event, body: %v. Full response: %v", eventBytes, res.String())
		return
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("Splunk Ingest API responded with unexpected status code %d while posting events. Full response: %v", res.StatusCode(), res.String())
	}
}
