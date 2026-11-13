// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extevents

import (
	"testing"

	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/stretchr/testify/assert"
)

func TestGetTargetTags(t *testing.T) {
	tags := getTargetTags(event_kit_api.ExperimentStepTargetExecution{
		ExecutionId:   42,
		ExperimentKey: "ADM-21",
		State:         event_kit_api.Completed,
	})

	assert.Equal(t, "42", tags["execution_id"])
	assert.Equal(t, "ADM-21", tags["experiment_key"])
	assert.Equal(t, "completed", tags["execution_state"])
}
