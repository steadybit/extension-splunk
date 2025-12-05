// check_test.go
package extslos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	actionApi "github.com/steadybit/action-kit/go/action_kit_api/v2"
)

// TestDescribe verifies that the action description is built as expected.
func TestDescribeCheck(t *testing.T) {
	action := &SloStateCheckAction{}
	desc := action.Describe()

	expectedID := fmt.Sprintf("%s.check", TargetType)
	if desc.Id != expectedID {
		t.Errorf("Describe() Id = %s; want %s", desc.Id, expectedID)
	}
	if desc.Label != "Check SLO Alerts" {
		t.Errorf("Describe() Label = %s; want %s", desc.Label, "Check SLO Alerts")
	}
	if desc.Description != "Check if the slo have active alerts." {
		t.Errorf("Describe() Description = %s; want %s", desc.Description, "Check if the slo have active alerts.")
	}
	if desc.TargetSelection == nil || desc.TargetSelection.TargetType != TargetType {
		t.Errorf("Describe() TargetSelection not set correctly")
	}
	if len(desc.Parameters) == 0 {
		t.Errorf("Expected at least one parameter in description")
	}
}

// TestPrepare verifies that Prepare populates the SloCheckState correctly.
func TestPrepare(t *testing.T) {
	action := &SloStateCheckAction{}
	state := action.NewEmptyState()
	req := actionApi.PrepareActionRequestBody{
		Target: &actionApi.Target{
			Attributes: map[string][]string{
				attributeID:   {"slo1"},
				attributeName: {"SLO One"},
			},
		},
		Config: map[string]interface{}{
			"duration":           30000.0, // milliseconds
			"expectedStateList":  breachAlertsTriggered,
			"stateCheckMode":     stateCheckModeAllTheTime,
			"checkNewAlertsOnly": true,
		},
	}

	_, err := action.Prepare(context.Background(), &state, req)
	if err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}
	if state.SloID != "slo1" {
		t.Errorf("Expected SloID 'slo1', got '%s'", state.SloID)
	}
	if state.SloName != "SLO One" {
		t.Errorf("Expected SloName 'SLO One', got '%s'", state.SloName)
	}
	if state.ExpectedState != breachAlertsTriggered {
		t.Errorf("Expected ExpectedState '%s', got '%s'", breachAlertsTriggered, state.ExpectedState)
	}
	if state.StateCheckMode != stateCheckModeAllTheTime {
		t.Errorf("Expected StateCheckMode '%s', got '%s'", stateCheckModeAllTheTime, state.StateCheckMode)
	}
	if !state.CheckNewAlertsOnly {
		t.Errorf("Expected CheckNewAlertsOnly to be true")
	}
	if state.Start.IsZero() || state.End.IsZero() {
		t.Errorf("Expected Start and End to be set")
	}
}

// newTestServer creates an HTTP test server for POST /v2/slo/search.
func newTestServer(respBody []byte, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/slo/search" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(respBody)
	}))
}

// TestStatus_AllTheTime_NoResult tests SLOCheckStatus for AllTheTime mode when no SLO is found.
func TestStatus_AllTheTime_NoResult(t *testing.T) {
	now := time.Now()
	state := SloCheckState{
		SloID:              "slo1",
		SloName:            "SLO One",
		CheckNewAlertsOnly: false,
		Start:              now.Add(-time.Minute),
		End:                now.Add(time.Minute),
		ExpectedState:      breachAlertsTriggered,
		StateCheckMode:     stateCheckModeAllTheTime,
	}
	// Simulate an empty response.
	respStruct := Response{
		Count:   0,
		Results: []Slo{},
	}
	respBytes, _ := json.Marshal(respStruct)
	ts := newTestServer(respBytes, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	statusResult, err := SLOCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("SLOCheckStatus returned error: %v", err)
	}
	if statusResult.Error == nil {
		t.Errorf("Expected error due to no SLO found, got nil")
	} else if !strings.Contains(statusResult.Error.Title, "was not found") {
		t.Errorf("Unexpected error message: %s", statusResult.Error.Title)
	}
}

// TestStatus_AllTheTime_Success tests SLOCheckStatus in AllTheTime mode when a matching SLO is found.
func TestStatus_AllTheTime_Success(t *testing.T) {
	now := time.Now()
	state := SloCheckState{
		SloID:              "slo1",
		SloName:            "SLO One",
		CheckNewAlertsOnly: false,
		Start:              now.Add(-time.Minute),
		End:                now.Add(time.Minute),
		ExpectedState:      breachAlertsTriggered,
		StateCheckMode:     stateCheckModeAllTheTime,
	}
	// Create a SLO that is returned.
	slo := Slo{
		ID:          "slo1",
		Name:        "SLO One",
		LastUpdated: now.Add(-2 * time.Minute).UnixMilli(),
	}
	respStruct := Response{
		Count:   1,
		Results: []Slo{slo},
	}
	respBytes, _ := json.Marshal(respStruct)
	ts := newTestServer(respBytes, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	statusResult, err := SLOCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("SLOCheckStatus returned error: %v", err)
	}
	if statusResult.Error != nil {
		t.Errorf("Expected no error, got: %v", statusResult.Error)
	}
	if len(*statusResult.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(*statusResult.Metrics))
	}
}

// TestStatus_AtLeastOnce_Success tests SLOCheckStatus in AtLeastOnce mode when a matching SLO is found.
func TestStatus_AtLeastOnce_Success(t *testing.T) {
	now := time.Now()
	state := SloCheckState{
		SloID:              "slo1",
		SloName:            "SLO One",
		CheckNewAlertsOnly: false,
		Start:              now.Add(-5 * time.Minute),
		End:                now.Add(-1 * time.Minute), // completed
		ExpectedState:      breachAlertsTriggered,
		StateCheckMode:     stateCheckModeAtLeastOnce,
	}
	slo := Slo{
		ID:          "slo1",
		Name:        "SLO One",
		LastUpdated: now.Add(-4 * time.Minute).UnixMilli(),
	}
	respStruct := Response{
		Count:   1,
		Results: []Slo{slo},
	}
	respBytes, _ := json.Marshal(respStruct)
	ts := newTestServer(respBytes, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	statusResult, err := SLOCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("SLOCheckStatus returned error: %v", err)
	}
	if statusResult.Error != nil {
		t.Errorf("Expected no error, got: %v", statusResult.Error)
	}
	if !state.StateCheckSuccess {
		t.Errorf("Expected StateCheckSuccess to be true")
	}
}

// TestStatus_AtLeastOnce_Failure tests SLOCheckStatus in AtLeastOnce mode when no matching SLO is found.
func TestStatus_AtLeastOnce_Failure(t *testing.T) {
	now := time.Now()
	state := SloCheckState{
		SloID:              "slo1",
		SloName:            "SLO One",
		CheckNewAlertsOnly: false,
		Start:              now.Add(-5 * time.Minute),
		End:                now.Add(-1 * time.Minute), // completed
		ExpectedState:      breachAlertsTriggered,
		StateCheckMode:     stateCheckModeAtLeastOnce,
	}
	respStruct := Response{
		Count:   0,
		Results: []Slo{},
	}
	respBytes, _ := json.Marshal(respStruct)
	ts := newTestServer(respBytes, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	statusResult, err := SLOCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("SLOCheckStatus returned error: %v", err)
	}
	if statusResult.Error == nil {
		t.Errorf("Expected error due to no matching SLO found, got nil")
	} else if !strings.Contains(statusResult.Error.Title, "didn't triggered") {
		t.Errorf("Unexpected error message: %s", statusResult.Error.Title)
	}
}

// TestStatus_ClientError tests that a client error is handled gracefully.
func TestStatus_ClientError(t *testing.T) {
	client := resty.New().SetTransport(&simulateClientErrorRoundTripper{})
	RestyClient = client

	state := SloCheckState{
		SloID:              "slo1",
		SloName:            "SLO One",
		CheckNewAlertsOnly: false,
		Start:              time.Now().Add(-5 * time.Minute),
		End:                time.Now().Add(5 * time.Minute),
		ExpectedState:      breachAlertsTriggered,
		StateCheckMode:     stateCheckModeAllTheTime,
	}
	_, err := SLOCheckStatus(context.Background(), &state, RestyClient)
	if err == nil {
		t.Errorf("Expected error due to client failure, got nil")
	}
}

// TestToMetric verifies that toMetric builds a metric correctly.
func TestToMetric(t *testing.T) {
	now := time.Now()
	// Use an existing Slo instance.
	slo := Slo{
		ID:   "slo1",
		Name: "SLO One",
	}
	// Case 1: expected state is noAlerts → metric state should be "info".
	metric := toMetric(noAlerts, slo, now)
	if metric == nil {
		t.Fatalf("toMetric returned nil")
	}
	expectedTooltip := fmt.Sprintf("SLO %s with %s found", slo.Name, noAlerts)
	if metric.Metric["tooltip"] != expectedTooltip {
		t.Errorf("Expected tooltip '%s', got '%s'", expectedTooltip, metric.Metric["tooltip"])
	}
	if !strings.Contains(metric.Metric["url"], slo.ID) {
		t.Errorf("Expected URL to contain '%s', got '%s'", slo.ID, metric.Metric["url"])
	}
	if metric.Metric["state"] != "info" {
		t.Errorf("Expected metric state 'info' for no alerts, got '%s'", metric.Metric["state"])
	}

	// Case 2: expected state is not noAlerts → metric state should be "danger".
	metric2 := toMetric(breachAlertsTriggered, slo, now)
	if metric2.Metric["state"] != "danger" {
		t.Errorf("Expected metric state 'danger' for breach alerts, got '%s'", metric2.Metric["state"])
	}
}
