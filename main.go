/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kit/extsignals"
	"github.com/steadybit/extension-splunk/config"
	"github.com/steadybit/extension-splunk/extdetectors"
	"github.com/steadybit/extension-splunk/extevents"
	"github.com/steadybit/extension-splunk/extslos"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
	"strings"
)

const (
	contentType         = "Content-Type"
	applciationJsonType = "application/json"
)

func main() {
	extlogging.InitZeroLog()

	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	exthealth.SetReady(false)
	exthealth.StartProbes(8084)

	config.ParseConfiguration()
	config.ValidateConfiguration()
	initRestyClient()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	discovery_kit_sdk.Register(extdetectors.NewDetectorDiscovery())
	action_kit_sdk.RegisterAction(extdetectors.NewDetectorStateCheckAction())
	extevents.RegisterEventListenerHandlers()

	discovery_kit_sdk.Register(extslos.NewSLODiscovery())
	action_kit_sdk.RegisterAction(extslos.NewSloStateCheckAction())

	extsignals.ActivateSignalHandlers()

	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8083,
	})
}

// ExtensionListResponse exists to merge the possible root path responses supported by the
// various extension kits. In this case, the response for ActionKit, DiscoveryKit and EventKit.
type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	event_kit_api.EventListenerList `json:",inline"`
	advice_kit_api.AdviceList       `json:",inline"`
}

func initRestyClient() {
	extdetectors.RestyClient = resty.New()
	extdetectors.RestyClient.SetBaseURL(strings.TrimRight(config.Config.ApiBaseUrl, "/"))
	extdetectors.RestyClient.SetHeader("Authorization", "Bearer "+config.Config.AccessToken)
	extdetectors.RestyClient.SetHeader(contentType, applciationJsonType)

	extslos.RestyClient = resty.New()
	extslos.RestyClient.SetBaseURL(strings.TrimRight(config.Config.ApiBaseUrl, "/"))
	extslos.RestyClient.SetHeader("Authorization", "Bearer "+config.Config.AccessToken)
	extslos.RestyClient.SetHeader(contentType, applciationJsonType)

	extevents.RestyClient = resty.New()
	extevents.RestyClient.SetBaseURL(strings.TrimRight(config.Config.IngestBaseUrl, "/"))
	extevents.RestyClient.SetHeader("X-SF-Token", config.Config.AccessToken)
	extevents.RestyClient.SetHeader(contentType, applciationJsonType)
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList:    action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
		EventListenerList: event_kit_api.EventListenerList{
			EventListeners: []event_kit_api.EventListener{
				{
					Method:   "POST",
					Path:     "/events/experiment-started",
					ListenTo: []string{"experiment.execution.created"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-completed",
					ListenTo: []string{"experiment.execution.completed", "experiment.execution.failed", "experiment.execution.canceled", "experiment.execution.errored"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-step-started",
					ListenTo: []string{"experiment.execution.step-started"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-step-completed",
					ListenTo: []string{"experiment.execution.step-completed", "experiment.execution.step-canceled", "experiment.execution.step-errored", "experiment.execution.step-failed"},
				},
			},
		},
	}
}
