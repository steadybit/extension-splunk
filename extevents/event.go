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
	"sync"
	"time"
)

func RegisterEventListenerHandlers() {
	exthttp.RegisterHttpHandler("/events/experiment-started", handle(onExperiment))
	exthttp.RegisterHttpHandler("/events/experiment-completed", handle(onExperimentCompleted))
	exthttp.RegisterHttpHandler("/events/experiment-step-started", handle(onExperimentStep))
	//exthttp.RegisterHttpHandler("/events/experiment-step-completed", handle(onExperimentStep))
	exthttp.RegisterHttpHandler("/events/experiment-target-started", handle(onExperimentTarget))
	exthttp.RegisterHttpHandler("/events/experiment-target-completed", handle(onExperimentTarget))
}

const (
	category = "USER_DEFINED"
)

var (
	stepExecutions = sync.Map{}
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

func onExperimentCompleted(event event_kit_api.EventRequestBody) (*Event, error) {
	stepExecutions.Range(func(key, value interface{}) bool {
		stepExecution := value.(event_kit_api.ExperimentStepExecution)
		if stepExecution.ExecutionId == event.ExperimentExecution.ExecutionId {
			log.Debug().Msgf("Delete step execution data for id %.0f", stepExecution.ExecutionId)
			stepExecutions.Delete(key)
		}
		return true
	})

	return onExperiment(event)
}

func onExperimentStep(event event_kit_api.EventRequestBody) (*Event, error) {
	tags := getEventBaseTags(event)
	maps.Copy(tags, getExecutionTags(event))
	maps.Copy(tags, getStepTags(*event.ExperimentStepExecution))

	stepExecutions.Store(event.ExperimentStepExecution.Id, *event.ExperimentStepExecution)

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

func getTargetTags(target event_kit_api.ExperimentStepTargetExecution) map[string]string {
	tags := make(map[string]string)

	tags["execution_id"] = fmt.Sprintf("%.0f", target.ExecutionId)
	tags["execution_id"] = target.ExperimentKey
	tags["execution_state"] = string(target.State)

	if target.StartedTime != nil {
		tags["started_time"] = target.StartedTime.Format(time.RFC3339)
	}

	if target.EndedTime != nil {
		tags["ended_time"] = target.EndedTime.Format(time.RFC3339)
	}

	return tags
}

func getTargetDimensions(target event_kit_api.ExperimentStepTargetExecution) map[string]string {
	dimensions := make(map[string]string)
	const clusterNameSteadybitAttribute = "k8s.cluster-name"

	if _, ok := target.TargetAttributes[clusterNameSteadybitAttribute]; ok {
		translateToSplunk(dimensions, target, clusterNameSteadybitAttribute, "k8s.cluster.name")
		translateToSplunk(dimensions, target, "k8s.namespace", "k8s.namespace.name")
		translateToSplunk(dimensions, target, "k8s.deployment", "k8s.deployment.name")
		translateToSplunk(dimensions, target, "k8s.pod.name", "clustername")
		translateToSplunk(dimensions, target, "k8s.container.name", "k8s.container.name")
	}

	getHostnameDimension(dimensions, target)
	translateToSplunk(dimensions, target, "container.id.stripped", "container.id")

	if _, ok := target.TargetAttributes["aws.region"]; ok {
		//AWS tags
		dimensions["cloud.provider"] = "aws"
		translateToSplunk(dimensions, target, "aws.region", "cloud.region")
		translateToSplunk(dimensions, target, "aws.zone", "cloud.availability_zone")
		translateToSplunk(dimensions, target, "aws.account", "cloud.account.id")
	}

	return dimensions
}

func getHostnameDimension(dimensions map[string]string, target event_kit_api.ExperimentStepTargetExecution) {
	const splunkHostDimensionName = "host.name"

	translateToSplunk(dimensions, target, "container.host", splunkHostDimensionName)
	translateToSplunk(dimensions, target, "container.host", splunkHostDimensionName)
	translateToSplunk(dimensions, target, "host.hostname", splunkHostDimensionName)
	translateToSplunk(dimensions, target, "application.hostname", splunkHostDimensionName)
}

func translateToSplunk(dimensions map[string]string, target event_kit_api.ExperimentStepTargetExecution, steadybitAttribute string, splunkDimension string) {
	if values, ok := target.TargetAttributes[steadybitAttribute]; ok {
		if (len(values)) == 1 {
			dimensions[splunkDimension] = values[0]
		}
	}
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

func onExperimentTarget(event event_kit_api.EventRequestBody) (*Event, error) {
	if event.ExperimentStepTargetExecution == nil {
		return nil, nil
	}

	var v, ok = stepExecutions.Load(event.ExperimentStepTargetExecution.StepExecutionId)
	if !ok {
		log.Warn().Msgf("Could not find step infos for step execution id %s", event.ExperimentStepTargetExecution.StepExecutionId)
		return nil, nil
	}
	stepExecution := v.(event_kit_api.ExperimentStepExecution)

	if stepExecution.ActionKind != nil && *stepExecution.ActionKind == event_kit_api.Attack {
		tags := getEventBaseTags(event)
		maps.Copy(tags, getExecutionTags(event))
		maps.Copy(tags, getStepTags(*event.ExperimentStepExecution))
		maps.Copy(tags, getTargetTags(*event.ExperimentStepTargetExecution))
		dimensions := getTargetDimensions(*event.ExperimentStepTargetExecution)

		stepExecutions.Store(event.ExperimentStepExecution.Id, *event.ExperimentStepExecution)

		return &Event{
			Category:   category,
			EventType:  "Steadybit_Event",
			Dimensions: dimensions,
			Properties: tags,
			Timestamp:  event.EventTime.UnixMilli(),
		}, nil
	}

	return nil, nil
}
