/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
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
	"github.com/steadybit/extension-scaffold/config"
	"github.com/steadybit/extension-scaffold/extadvice/robot_maintenance"
	"github.com/steadybit/extension-scaffold/extevents"
	"github.com/steadybit/extension-scaffold/extrobots"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
)

func main() {
	// Most Steadybit extensions leverage zerolog. To encourage persistent logging setups across extensions,
	// you may leverage the extlogging package to initialize zerolog. Among others, this package supports
	// configuration of active log levels and the log format (JSON or plain text).
	//
	// Example
	//  - to activate JSON logging, set the environment variable STEADYBIT_LOG_FORMAT="json"
	//  - to set the log level to debug, set the environment variable STEADYBIT_LOG_LEVEL="debug"
	extlogging.InitZeroLog()

	// Build information is set at compile-time. This line writes the build information to the log.
	// The information is mostly handy for debugging purposes.
	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	// Most extensions require some form of configuration. These calls exist to parse and validate the
	// configuration obtained from environment variables.
	config.ParseConfiguration()
	config.ValidateConfiguration()

	//This will start /health/liveness and /health/readiness endpoints on port 8081 for use with kubernetes
	//The port can be configured using the STEADYBIT_EXTENSION_HEALTH_PORT environment variable
	exthealth.SetReady(false)
	exthealth.StartProbes(8081)

	// This call registers a handler for the extension's root path. This is the path initially accessed
	// by the Steadybit agent to obtain the extension's capabilities.
	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	// This is a section you will most likely want to change: The registration of HTTP handlers
	// for your extension. You might want to change these because the names do not fit, or because
	// you do not have a need for all of them.
	discovery_kit_sdk.Register(extrobots.NewRobotDiscovery())
	action_kit_sdk.RegisterAction(extrobots.NewLogAction())
	extevents.RegisterEventListenerHandlers()

	// Register the handler for the advice endpoint
	exthttp.RegisterHttpHandler("/advice/robot-maintenance", exthttp.GetterAsHandler(robot_maintenance.GetAdviceDescriptionRobotMaintenance))

	//This will install a signal handlder, that will stop active actions when receiving a SIGURS1, SIGTERM or SIGINT
	extsignals.ActivateSignalHandlers()

	//This will register the coverage endpoints for the extension (used by action_kit_test)
	action_kit_sdk.RegisterCoverageEndpoints()

	//This will switch the readiness state of the application to true.
	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		// This is the default port under which your extension is accessible.
		// The port can be configured externally through the
		// STEADYBIT_EXTENSION_PORT environment variable.
		// We suggest that you keep port 8080 as the default.
		Port: 8080,
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

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		// See this document to learn more about the action list:
		// https://github.com/steadybit/action-kit/blob/main/docs/action-api.md#action-list
		ActionList: action_kit_sdk.GetActionList(),

		// See this document to learn more about the discovery list:
		// https://github.com/steadybit/discovery-kit/blob/main/docs/discovery-api.md#index-response
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),

		// See this document to learn more about the event listener list:
		// https://github.com/steadybit/event-kit/blob/main/docs/event-api.md#event-listeners-list
		EventListenerList: extevents.GetEventListenerList(),

		// See this document to learn more about the advice list:
		// https://github.com/steadybit/advice-kit/blob/main/docs/advice-api.md#index-response
		AdviceList: advice_kit_api.AdviceList{
			Advice: getAdviceRefs(),
		},
	}
}

func getAdviceRefs() []advice_kit_api.DescribingEndpointReference {
	var refs []advice_kit_api.DescribingEndpointReference
	refs = make([]advice_kit_api.DescribingEndpointReference, 0)
	for _, adviceId := range config.Config.ActiveAdviceList {
		// Maintenance advice
		if adviceId == "*" || adviceId == robot_maintenance.RobotMaintenanceID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/robot-maintenance",
			})
		}
	}
	return refs
}
