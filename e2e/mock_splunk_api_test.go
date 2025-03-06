/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package e2e

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-splunk/extdetectors"
	"net"
	"net/http"
	"net/http/httptest"
)

type mockServer struct {
	http  *httptest.Server
	state string
}

func createMockSplunkApiServer() *mockServer {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	mux := http.NewServeMux()

	server := httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")

	mock := &mockServer{http: &server, state: "CLEAR"}
	mux.Handle("GET /v2/detector", handler(mock.getDetectors))
	mux.Handle("GET /v2/detector/GlHqGZmCkAE/incidents", handler(mock.getDetectorIncidents))
	return mock
}

func handler[T any](getter func() T) http.Handler {
	return exthttp.PanicRecovery(exthttp.LogRequest(exthttp.GetterAsHandler(getter)))
}

func (m *mockServer) getDetectors() extdetectors.Response {
	if m.state == "STATUS-500" {
		panic("status 500")
	}

	return extdetectors.Response{
		Count: 2,
		Results: []extdetectors.Detector{
			{
				AuthorizedWriters:            extdetectors.AuthorizedWriters{},
				AutoOptimizationDisabled:     false,
				Created:                      1741018151464,
				Creator:                      "AAAAAAAAAAA",
				CustomProperties:             nil,
				Description:                  "Alerts when custom MTS usage percentage is above threshold",
				DetectorOrigin:               "AutoDetect",
				ID:                           "GlHqGZmCkAE",
				ImportQualifiers:             nil,
				LabelResolutions:             nil,
				LastOptimized:                0,
				LastUpdated:                  1741086004665,
				LastUpdatedBy:                "AAAAAAAAAAA",
				MaxDelay:                     0,
				MinDelay:                     nil,
				Name:                         "Splunk operational - Custom MTS usage is expected to reach the limit",
				OverMTSLimit:                 false,
				PackageSpecifications:        "",
				ProgramText:                  "",
				Rules:                        nil,
				SFMetricsInObjectProgramText: nil,
				Status:                       "ACTIVE",
				Tags:                         nil,
				Teams:                        nil,
				Timezone:                     nil,
				VisualizationOptions:         extdetectors.VisualizationOptions{},
			},
		},
	}
}

func (m *mockServer) getDetectorIncidents() []extdetectors.Incident {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extdetectors.Incident{
		{
			Active:                      true,
			AnomalyState:                "ANOMALOUS",
			AnomalyStateUpdateTimestamp: 1741019530000,
			DetectLabel:                 "APM - Sudden change in service request rate",
			DetectorId:                  "GlHqGZmCkAE",
			DetectorName:                "Splunk operational - Custom MTS usage is expected to reach the limit",
			DisplayBody:                 "test",
			Events:                      nil,
			IncidentId:                  "GlHA2unCgAA",
			IsMuted:                     false,
			Severity:                    "Critical",
			TriggeredNotificationSent:   false,
			TriggeredWhileMuted:         false,
		},
	}
}
