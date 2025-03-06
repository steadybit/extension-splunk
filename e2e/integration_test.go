/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	actValidate "github.com/steadybit/action-kit/go/action_kit_test/validate"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	server := createMockSplunkApiServer()
	defer server.http.Close()
	split := strings.SplitAfter(server.http.URL, ":")
	port := split[len(split)-1]

	extlogging.InitZeroLog()

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-splunk",
		Port: 8083,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", fmt.Sprintf("splunk.apiBaseUrl=http://host.minikube.internal:%s", port),
				"--set", "logging.level=trace",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "validate discovery",
			Test: validateDiscovery,
		},
		{
			Name: "test discovery",
			Test: testDiscovery,
		},
		{
			Name: "validate Actions",
			Test: validateActions,
		},
		{
			Name: "detector check meets expectations",
			Test: testAlertRuleCheckNormal(server, "normal", "ANOMALOUS", ""),
		},
	})
}

func testAlertRuleCheckNormal(server *mockServer, status, expectedState string, wantedActionStatus action_kit_api.ActionKitErrorStatus) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
		target := &action_kit_api.Target{
			Name: "test_anomalous",
			Attributes: map[string][]string{
				"splunk.detector.status":         {"ACTIVE"},
				"splunk.detector.name":           {"Splunk operational - Custom MTS usage is expected to reach the limit"},
				"splunk.detector.creator":        {"AAAAAAAAAAA"},
				"splunk.detector.description":    {"Alerts when custom MTS usage percentage is above threshold"},
				"splunk.detector.id":             {"GlHqGZmCkAE"},
				"splunk.detector.detectorOrigin": {"AutoDetect"},
			},
		}

		config := struct {
			Duration      int    `json:"duration"`
			ExpectedState string `json:"expectedState"`
		}{Duration: 1_000, ExpectedState: expectedState}

		server.state = status
		action, err := e.RunAction("com.steadybit.extension_splunk.detector.check", target, config, &action_kit_api.ExecutionContext{})
		require.NoError(t, err)
		defer func() { _ = action.Cancel() }()

		err = action.Wait()
		if wantedActionStatus == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, fmt.Sprintf("[%s]", wantedActionStatus))
		}
	}
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func testDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	target, err := e2e.PollForTarget(ctx, e, "com.steadybit.extension_splunk.detector", func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "splunk.detector.id", "GlHqGZmCkAE")
	})
	require.NoError(t, err)
	assert.Equal(t, target.TargetType, "com.steadybit.extension_splunk.detector")
	assert.Equal(t, target.Attributes["splunk.detector.id"], []string{"GlHqGZmCkAE"})
	assert.Equal(t, target.Attributes["splunk.detector.description"], []string{"Alerts when custom MTS usage percentage is above threshold"})
	assert.Equal(t, target.Attributes["splunk.detector.status"], []string{"ACTIVE"})
}

func validateActions(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, actValidate.ValidateEndpointReferences("/", e.Client))
}
