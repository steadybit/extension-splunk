package extdetectors

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrepareExtractsState(t *testing.T) {
	// Given
	request := extutil.JsonMangle(action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":          1000 * 60,
			"expectedStateList": []string{"MANUALLY_RESOLVED"},
		},
		Target: &action_kit_api.Target{
			Attributes: map[string][]string{
				"splunk.detector.status":         {"ACTIVE"},
				"splunk.detector.name":           {"Splunk operational - Custom MTS usage is expected to reach the limit"},
				"splunk.detector.creator":        {"AAAAAAAAAAA"},
				"splunk.detector.description":    {"Alerts when custom MTS usage percentage is above threshold"},
				"splunk.detector.id":             {"GlHqGZmCkAE"},
				"splunk.detector.detectorOrigin": {"AutoDetect"},
			},
		},
		ExecutionContext: extutil.Ptr(action_kit_api.ExecutionContext{
			ExperimentUri: extutil.Ptr("<uri-to-experiment>"),
			ExecutionUri:  extutil.Ptr("<uri-to-execution>"),
		}),
	})
	action := DetectorStateCheckAction{}
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, result)
	require.Nil(t, err)
	require.Equal(t, "GlHqGZmCkAE", state.DetectorId)
	require.Equal(t, "Splunk operational - Custom MTS usage is expected to reach the limit", state.DetectorName)
	require.Equal(t, "[MANUALLY_RESOLVED]", state.ExpectedState)
}
