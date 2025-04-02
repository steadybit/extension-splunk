// check_test.go
package extdetectors

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

// dummyPrepareRequest creates a dummy PrepareActionRequestBody with a target and config.
func dummyPrepareRequest(duration float64, expectedState, stateCheckMode string, checkNewIncidentsOnly bool) actionApi.PrepareActionRequestBody {
	return actionApi.PrepareActionRequestBody{
		Target: &actionApi.Target{
			Attributes: map[string][]string{
				attributeID:   {"detector1"},
				attributeName: {"Detector One"},
			},
		},
		Config: map[string]interface{}{
			"duration":              duration,
			"expectedStateList":     expectedState,
			"stateCheckMode":        stateCheckMode,
			"checkNewIncidentsOnly": checkNewIncidentsOnly,
		},
	}
}

// --- Tests for Describe ---

func TestDescribe(t *testing.T) {
	action := &DetectorStateCheckAction{}
	desc := action.Describe()

	expectedID := fmt.Sprintf("%s.check", TargetType)
	if desc.Id != expectedID {
		t.Errorf("Describe() Id = %s; want %s", desc.Id, expectedID)
	}
	if desc.Label != "Check Detector Incidents" {
		t.Errorf("Describe() Label = %s; want %s", desc.Label, "Check Detector Incidents")
	}
	// Check that at least one parameter is defined.
	if len(desc.Parameters) == 0 {
		t.Errorf("Expected at least one parameter in description")
	}
}

// --- Tests for Prepare ---

func TestPrepare(t *testing.T) {
	action := &DetectorStateCheckAction{}
	state := action.NewEmptyState()
	req := dummyPrepareRequest(30000, Anomalous, stateCheckModeAllTheTime, true)

	// Call Prepare. (We ignore the returned PrepareResult as it is nil.)
	_, err := action.Prepare(context.Background(), &state, req)
	if err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	// Verify that target attributes are set.
	if state.DetectorId != "detector1" {
		t.Errorf("Expected DetectorId 'detector1', got '%s'", state.DetectorId)
	}
	if state.DetectorName != "Detector One" {
		t.Errorf("Expected DetectorName 'Detector One', got '%s'", state.DetectorName)
	}
	// Check that Start and End are set (End should be roughly Start+duration).
	if state.Start.IsZero() {
		t.Errorf("Start time not set")
	}
	expectedEnd := state.Start.Add(time.Millisecond * 30000)
	if state.End.Sub(expectedEnd) > time.Second {
		t.Errorf("End time not set correctly; expected around %v, got %v", expectedEnd, state.End)
	}
	if state.ExpectedState != Anomalous {
		t.Errorf("Expected ExpectedState '%s', got '%s'", Anomalous, state.ExpectedState)
	}
	if state.StateCheckMode != stateCheckModeAllTheTime {
		t.Errorf("Expected StateCheckMode '%s', got '%s'", stateCheckModeAllTheTime, state.StateCheckMode)
	}
}

// --- Tests for Describe ---

func TestDescribeCheck(t *testing.T) {
	action := &DetectorStateCheckAction{}
	desc := action.Describe()

	expectedID := fmt.Sprintf("%s.check", TargetType)
	if desc.Id != expectedID {
		t.Errorf("Describe() Id = %s; want %s", desc.Id, expectedID)
	}
	if desc.Label != "Check Detector Incidents" {
		t.Errorf("Describe() Label = %s; want %s", desc.Label, "Check Detector Incidents")
	}
	// Check that at least one parameter is defined.
	if len(desc.Parameters) == 0 {
		t.Errorf("Expected at least one parameter in description")
	}
}

// --- Tests for DetectorCheckStatus ---

// Helper to create a test server that returns a JSON array of incidents.
func newTestServer(incidents []Incident, statusCode int) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v2/detector/") || !strings.HasSuffix(r.URL.Path, "/incidents") {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(incidents)
	})
	return httptest.NewServer(handler)
}

func TestStatus_AllTheTime_Success(t *testing.T) {
	// Simulate an incident that matches the expected state.
	now := time.Now()
	incident := Incident{
		AnomalyState:                Anomalous,
		AnomalyStateUpdateTimestamp: now.Add(-time.Minute).UnixMilli(),
	}
	ts := newTestServer([]Incident{incident}, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	// Create a state with End in the future.
	state := DetectorCheckState{
		DetectorId:            "detector1",
		DetectorName:          "Detector One",
		CheckNewIncidentsOnly: false,
		Start:                 time.Now().Add(-2 * time.Minute),
		End:                   time.Now().Add(2 * time.Minute),
		ExpectedState:         Anomalous,
		StateCheckMode:        stateCheckModeAllTheTime,
	}

	statusResult, err := DetectorCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("DetectorCheckStatus returned error: %v", err)
	}
	if statusResult.Error != nil {
		t.Errorf("Expected no error, got error: %v", statusResult.Error)
	}
	if len(*statusResult.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(*statusResult.Metrics))
	}
	// Check that the metric has a state mapping of "danger" for Anomalous.
	metric := (*statusResult.Metrics)[0]
	if metric.Metric["state"] != "danger" {
		t.Errorf("Expected metric state 'danger', got '%s'", metric.Metric["state"])
	}
}

func TestStatus_AllTheTime_Failure(t *testing.T) {
	// Simulate an incident that does not match the expected state.
	now := time.Now()
	incident := Incident{
		AnomalyState:                Ok, // not matching expected Anomalous
		AnomalyStateUpdateTimestamp: now.Add(-time.Minute).UnixMilli(),
	}
	ts := newTestServer([]Incident{incident}, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	state := DetectorCheckState{
		DetectorId:            "detector1",
		DetectorName:          "Detector One",
		CheckNewIncidentsOnly: false,
		Start:                 time.Now().Add(-2 * time.Minute),
		End:                   time.Now().Add(2 * time.Minute),
		ExpectedState:         Anomalous,
		StateCheckMode:        stateCheckModeAllTheTime,
	}

	statusResult, err := DetectorCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("DetectorCheckStatus returned error: %v", err)
	}
	if statusResult.Error == nil {
		t.Errorf("Expected error due to mismatching incident state, but got nil")
	} else if !strings.Contains(statusResult.Error.Title, "has state") {
		t.Errorf("Unexpected error message: %s", statusResult.Error.Title)
	}
}

func TestStatus_AtLeastOnce_Success(t *testing.T) {
	// Simulate an incident that matches expected state.
	now := time.Now()
	incident := Incident{
		AnomalyState:                Anomalous,
		AnomalyStateUpdateTimestamp: now.Add(-time.Minute).UnixMilli(),
	}
	ts := newTestServer([]Incident{incident}, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	// Set End in the past to mark completion.
	state := DetectorCheckState{
		DetectorId:            "detector1",
		DetectorName:          "Detector One",
		CheckNewIncidentsOnly: false,
		Start:                 time.Now().Add(-5 * time.Minute),
		End:                   time.Now().Add(-1 * time.Minute),
		ExpectedState:         Anomalous,
		StateCheckMode:        stateCheckModeAtLeastOnce,
	}

	statusResult, err := DetectorCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("DetectorCheckStatus returned error: %v", err)
	}
	if statusResult.Error != nil {
		t.Errorf("Expected no error as at least one incident matched, got error: %v", statusResult.Error)
	}
	if !state.StateCheckSuccess {
		t.Errorf("Expected StateCheckSuccess to be true")
	}
}

func TestStatus_AtLeastOnce_Failure(t *testing.T) {
	// Simulate an incident that does not match the expected state.
	now := time.Now()
	incident := Incident{
		AnomalyState:                Ok, // does not match expected
		AnomalyStateUpdateTimestamp: now.Add(-time.Minute).UnixMilli(),
	}
	ts := newTestServer([]Incident{incident}, http.StatusOK)
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	// End in the past.
	state := DetectorCheckState{
		DetectorId:            "detector1",
		DetectorName:          "Detector One",
		CheckNewIncidentsOnly: false,
		Start:                 time.Now().Add(-5 * time.Minute),
		End:                   time.Now().Add(-1 * time.Minute),
		ExpectedState:         Anomalous,
		StateCheckMode:        stateCheckModeAtLeastOnce,
	}

	statusResult, err := DetectorCheckStatus(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("DetectorCheckStatus returned error: %v", err)
	}
	if statusResult.Error == nil {
		t.Errorf("Expected error since no incident matched expected state, but got nil")
	} else if !strings.Contains(statusResult.Error.Title, "didn't have status") {
		t.Errorf("Unexpected error message: %s", statusResult.Error.Title)
	}
}

func TestStatus_ClientError(t *testing.T) {
	// Create a resty client that always returns an error.
	client := resty.New().SetTransport(&errorRoundTripper{})
	RestyClient = client

	state := DetectorCheckState{
		DetectorId:            "detector1",
		DetectorName:          "Detector One",
		CheckNewIncidentsOnly: false,
		Start:                 time.Now().Add(-5 * time.Minute),
		End:                   time.Now().Add(5 * time.Minute),
		ExpectedState:         Anomalous,
		StateCheckMode:        stateCheckModeAllTheTime,
	}

	_, err := DetectorCheckStatus(context.Background(), &state, RestyClient)
	if err == nil {
		t.Errorf("Expected error due to client failure, got nil")
	}
}

// --- Test for toMetric ---

func TestToMetric(t *testing.T) {
	now := time.Now()
	incident := Incident{
		AnomalyState:                Anomalous,
		AnomalyStateUpdateTimestamp: now.UnixMilli(),
	}
	metric := toMetric("detector1", "Detector One", incident, now)
	if metric == nil {
		t.Fatalf("toMetric returned nil")
	}
	expectedTooltip := fmt.Sprintf("Detector incident state is: %s", incident.AnomalyState)
	if metric.Metric["tooltip"] != expectedTooltip {
		t.Errorf("Expected tooltip '%s', got '%s'", expectedTooltip, metric.Metric["tooltip"])
	}
	// Check URL contains the detector ID.
	if !strings.Contains(metric.Metric["url"], "detector1") {
		t.Errorf("Expected URL to contain 'detector1', got '%s'", metric.Metric["url"])
	}
	// For Anomalous, our mapping should yield "danger".
	if metric.Metric["state"] != "danger" {
		t.Errorf("Expected metric state 'danger' for anomaly state '%s', got '%s'", incident.AnomalyState, metric.Metric["state"])
	}
}
