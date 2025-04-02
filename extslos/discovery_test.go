// extslos_test.go
package extslos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/steadybit/extension-splunk/config"
)

// TestDescribe verifies that the discovery description is built as expected.
func TestDescribeDiscovery(t *testing.T) {
	d := &sloDiscovery{}
	desc := d.Describe()

	if desc.Id != TargetType {
		t.Errorf("Describe() Id = %s; want %s", desc.Id, TargetType)
	}
	if desc.Discover.CallInterval == nil || *desc.Discover.CallInterval != "1m" {
		t.Errorf("Describe() CallInterval = %v; want 1m", desc.Discover.CallInterval)
	}
}

// TestDescribeTarget checks the target description returned by the discovery.
func TestDescribeTarget(t *testing.T) {
	d := &sloDiscovery{}
	td := d.DescribeTarget()

	if td.Id != TargetType {
		t.Errorf("DescribeTarget() Id = %s; want %s", td.Id, TargetType)
	}
	if td.Label.One != "Splunk SLO" || td.Label.Other != "Splunk SLOs" {
		t.Errorf("DescribeTarget() Label = %+v; want {One: 'Splunk SLO', Other: 'Splunk SLOs'}", td.Label)
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
	d := &sloDiscovery{}
	attrs := d.DescribeAttributes()
	expected := []string{attributeID, attributeName, attributeIndicator, attributeCreator}

	if len(attrs) != len(expected) {
		t.Errorf("DescribeAttributes() length = %d; want %d", len(attrs), len(expected))
	}
	for i, attr := range attrs {
		if attr.Attribute != expected[i] {
			t.Errorf("DescribeAttributes() attr[%d] = %s; want %s", i, attr.Attribute, expected[i])
		}
	}
}

// TestDiscoverTargets_ValidResponse simulates a valid Splunk response with one SLO.
func TestDiscoverTargets_ValidResponse(t *testing.T) {
	// Create a test HTTP server that returns a valid JSON response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/slo/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// The JSON should match the expected structure of the Response type.
		w.Write([]byte(`{"Results": [{"ID": "slo1", "Name": "Test SLO", "Indicator": "up", "Creator": "user1"}]}`))
	}))
	defer ts.Close()

	// Create a resty client that points to the test server.
	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	d := &sloDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("DiscoverTargets returned %d targets; want 1", len(targets))
	} else {
		target := targets[0]
		if target.Id != "slo1" {
			t.Errorf("Target.Id = %s; want %s", target.Id, "slo1")
		}
		if target.TargetType != TargetType {
			t.Errorf("Target.TargetType = %s; want %s", target.TargetType, TargetType)
		}
		if target.Label != "Test SLO" {
			t.Errorf("Target.Label = %s; want %s", target.Label, "Test SLO")
		}
		attrs := target.Attributes
		if attr, ok := attrs[attributeID]; !ok || attr[0] != "slo1" {
			t.Errorf("Attribute %s = %v; want ['slo1']", attributeID, attr)
		}
		if attr, ok := attrs[attributeName]; !ok || attr[0] != "Test SLO" {
			t.Errorf("Attribute %s = %v; want ['Test SLO']", attributeName, attr)
		}
		if attr, ok := attrs[attributeIndicator]; !ok || attr[0] != "up" {
			t.Errorf("Attribute %s = %v; want ['up']", attributeIndicator, attr)
		}
		if attr, ok := attrs[attributeCreator]; !ok || attr[0] != "user1" {
			t.Errorf("Attribute %s = %v; want ['user1']", attributeCreator, attr)
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

	d := &sloDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("DiscoverTargets returned %d targets; want 0", len(targets))
	}
}

// TestDiscoverTargets_404Response tests that a 404 response is handled (returning an empty target slice).
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

	d := &sloDiscovery{}
	targets, err := d.DiscoverTargets(context.Background())
	if err != nil {
		t.Errorf("DiscoverTargets returned error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("DiscoverTargets returned %d targets; want 0", len(targets))
	}
}

// errorRoundTripper is used to simulate an HTTP client error.
type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("forced error")
}

// TestDiscoverTargets_ClientError tests that an error during the HTTP call is handled gracefully.
func TestDiscoverTargets_ClientError(t *testing.T) {
	client := resty.New().SetTransport(&errorRoundTripper{})
	RestyClient = client

	d := &sloDiscovery{}
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
	// Create a test server that returns one SLO.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Results": [{"ID": "slo1", "Name": "Test SLO", "Indicator": "up", "Creator": "user1"}]}`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	RestyClient = client

	// Backup the original exclusion configuration and set an exclusion for the "splunk.slo.name" attribute.
	originalExcludes := config.Config.DiscoveryAttributesExcludesSLO
	config.Config.DiscoveryAttributesExcludesSLO = []string{attributeName}

	d := &sloDiscovery{}
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

	// Restore the original exclusion configuration.
	config.Config.DiscoveryAttributesExcludesSLO = originalExcludes
}
