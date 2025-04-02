package extslos

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
			"expectedStateList": []string{"breach alerts"},
		},
		Target: &action_kit_api.Target{
			Attributes: map[string][]string{
				attributeName:      {"SLO frontend:call"},
				attributeCreator:   {"AAAAAAAAAAA"},
				attributeID:        {"GlHE"},
				attributeIndicator: {"app"},
			},
		},
		ExecutionContext: extutil.Ptr(action_kit_api.ExecutionContext{
			ExperimentUri: extutil.Ptr("<uri-to-experiment>"),
			ExecutionUri:  extutil.Ptr("<uri-to-execution>"),
		}),
	})
	action := SloStateCheckAction{}
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, result)
	require.Nil(t, err)
	require.Equal(t, "GlHE", state.SloID)
	require.Equal(t, "SLO frontend:call", state.SloName)
	require.Equal(t, "[breach alerts]", state.ExpectedState)
}
