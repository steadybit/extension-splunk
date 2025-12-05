// discovery_test.go
package extdetectors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/steadybit/extension-splunk/config"
)

// TestDescribeTarget checks the target description returned by the discovery.
func TestDescribeTarget(t *testing.T) {
	d := &detectorDiscovery{}
	td := d.DescribeTarget()

	if td.Id != TargetType {
		t.Errorf("DescribeTarget() Id = %s; want %s", td.Id, TargetType)
	}
	if td.Label.One != "Splunk detector" || td.Label.Other != "Splunk detectors" {
		t.Errorf("DescribeTarget() Label = %+v; want {One: 'Splunk detector', Other: 'Splunk detectors'}", td.Label)
	}
	if td.Category == nil || *td.Category != "monitoring" {
		t.Errorf("DescribeTarget() Category = %v; want 'monitoring'", td.Category)
	}
	if td.Icon == nil || *td.Icon != targetIcon {
		t.Errorf("DescribeTarget() Icon = %v; want %s", td.Icon, targetIcon)
	}
}

// TestDescribeAttributes ensures that the correct attribute descriptions are returned.
func TestDescribeAttributes(t *testing.T) {
	d := &detectorDiscovery{}
	attrs := d.DescribeAttributes()
	expected := []string{
		attributeID,
		attributeName,
		attributeDescription,
		attributeStatus,
		attributeCreator,
		attributeDetectorOrigin,
	}

	if len(attrs) != len(expected) {
		t.Errorf("DescribeAttributes() length = %d; want %d", len(attrs), len(expected))
	}
	for i, attr := range attrs {
		if attr.Attribute != expected[i] {
			t.Errorf("DescribeAttributes() attr[%d] = %s; want %s", i, attr.Attribute, expected[i])
		}
	}
}

// TestDiscoverTargets_ValidResponse simulates a valid Splunk response with one detector.
func TestDiscoverTargets_ValidResponse(t *testing.T) {
	// Create a test HTTP server that returns a valid JSON response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/detector" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// The JSON should match the expected structure of the Response type.
		w.Write([]byte(`{"Results": [{"ID": "det1", "Name": "Test Detector", "Description": "Test description", "Status": "active", "Creator": "user1", "DetectorOrigin": "origin1"}]}`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	d := &detectorDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("DiscoverTargets returned %d targets; want 1", len(targets))
	} else {
		target := targets[0]
		if target.Id != "det1" {
			t.Errorf("Target.Id = %s; want %s", target.Id, "det1")
		}
		if target.TargetType != TargetType {
			t.Errorf("Target.TargetType = %s; want %s", target.TargetType, TargetType)
		}
		if target.Label != "Test Detector" {
			t.Errorf("Target.Label = %s; want %s", target.Label, "Test Detector")
		}
		attrs := target.Attributes
		if attr, ok := attrs[attributeID]; !ok || attr[0] != "det1" {
			t.Errorf("Attribute %s = %v; want ['det1']", attributeID, attr)
		}
		if attr, ok := attrs[attributeName]; !ok || attr[0] != "Test Detector" {
			t.Errorf("Attribute %s = %v; want ['Test Detector']", attributeName, attr)
		}
		if attr, ok := attrs[attributeDescription]; !ok || attr[0] != "Test description" {
			t.Errorf("Attribute %s = %v; want ['Test description']", attributeDescription, attr)
		}
		if attr, ok := attrs[attributeStatus]; !ok || attr[0] != "active" {
			t.Errorf("Attribute %s = %v; want ['active']", attributeStatus, attr)
		}
		if attr, ok := attrs[attributeCreator]; !ok || attr[0] != "user1" {
			t.Errorf("Attribute %s = %v; want ['user1']", attributeCreator, attr)
		}
		if attr, ok := attrs[attributeDetectorOrigin]; !ok || attr[0] != "origin1" {
			t.Errorf("Attribute %s = %v; want ['origin1']", attributeDetectorOrigin, attr)
		}
	}
}

// TestDiscoverTargets_UnexpectedStatus tests the behavior when the Splunk API returns an unexpected status code.
func TestDiscoverTargets_UnexpectedStatus(t *testing.T) {
	// Create a test server that returns a 500 error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	d := &detectorDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("DiscoverTargets returned %d targets; want 0", len(targets))
	}
}

// TestDiscoverTargets_404Response tests that a 404 response is handled correctly (returning an empty slice).
func TestDiscoverTargets_404Response(t *testing.T) {
	// Create a test server that returns a 404 with an empty JSON result.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"Results": []}`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	d := &detectorDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("DiscoverTargets returned %d targets; want 0", len(targets))
	}
}

// simulateClientErrorRoundTripper is used to simulate an HTTP client error.
type simulateClientErrorRoundTripper struct{}

func (e *simulateClientErrorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("forced error")
}

// TestDiscoverTargets_ClientError tests that an error during the HTTP call is handled gracefully.
func TestDiscoverTargets_ClientError(t *testing.T) {
	client := resty.New().SetTransport(&simulateClientErrorRoundTripper{})
	RestyClient = client

	d := &detectorDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("DiscoverTargets returned %d targets; want 0", len(targets))
	}
}

// TestDiscoverTargets_AttributeExclusion tests that the attribute exclusion configuration is applied.
func TestDiscoverTargets_AttributeExclusion(t *testing.T) {
	// Create a test server that returns one detector.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Results": [{"ID": "det1", "Name": "Test Detector", "Description": "Test description", "Status": "active", "Creator": "user1", "DetectorOrigin": "origin1"}]}`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	// Backup the original exclusion configuration and set an exclusion for the "splunk.detector.name" attribute.
	originalExcludes := config.Config.DiscoveryAttributesExcludesDetector
	config.Config.DiscoveryAttributesExcludesDetector = []string{attributeName}
	defer func() {
		config.Config.DiscoveryAttributesExcludesDetector = originalExcludes
	}()

	d := &detectorDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("DiscoverTargets returned %d targets; want 1", len(targets))
	} else {
		attrs := targets[0].Attributes
		if _, ok := attrs[attributeName]; ok {
			t.Errorf("Attribute %s should be excluded", attributeName)
		}
		if _, ok := attrs[attributeID]; !ok {
			t.Errorf("Attribute %s should be present", attributeID)
		}
	}
}
