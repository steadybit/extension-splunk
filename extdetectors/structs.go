/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdetectors

type Response struct {
	Count   int        `json:"count"`
	Results []Detector `json:"results"`
}

type Detector struct {
	AuthorizedWriters            AuthorizedWriters      `json:"authorizedWriters"`
	AutoOptimizationDisabled     bool                   `json:"autoOptimizationDisabled"`
	Created                      int64                  `json:"created"`
	Creator                      string                 `json:"creator"`
	CustomProperties             map[string]interface{} `json:"customProperties"`
	Description                  string                 `json:"description"`
	DetectorOrigin               string                 `json:"detectorOrigin"`
	ID                           string                 `json:"id"`
	ImportQualifiers             []ImportQualifier      `json:"importQualifiers"`
	LabelResolutions             map[string]int         `json:"labelResolutions"`
	LastOptimized                int64                  `json:"lastOptimized"`
	LastUpdated                  int64                  `json:"lastUpdated"`
	LastUpdatedBy                string                 `json:"lastUpdatedBy"`
	MaxDelay                     int64                  `json:"maxDelay"`
	MinDelay                     *int64                 `json:"minDelay"` // pointer to handle null values
	Name                         string                 `json:"name"`
	OverMTSLimit                 bool                   `json:"overMTSLimit"`
	PackageSpecifications        string                 `json:"packageSpecifications"`
	ProgramText                  string                 `json:"programText"`
	Rules                        []Rule                 `json:"rules"`
	SFMetricsInObjectProgramText []string               `json:"sf_metricsInObjectProgramText"`
	Status                       string                 `json:"status"`
	Tags                         []string               `json:"tags"`
	Teams                        []string               `json:"teams"`
	Timezone                     *string                `json:"timezone"` // pointer to handle null values
	VisualizationOptions         VisualizationOptions   `json:"visualizationOptions"`
}

type AuthorizedWriters struct {
	Teams []string `json:"teams"`
	Users []string `json:"users"`
}

// ImportQualifier represents each import qualifier.
type ImportQualifier struct {
	Filters []Filter `json:"filters"`
	Metric  string   `json:"metric"`
}

type Filter struct {
	Not      bool     `json:"not"`
	Property string   `json:"property"`
	Values   []string `json:"values"`
}

type Rule struct {
	Description          string        `json:"description"`
	DetectLabel          string        `json:"detectLabel"`
	Disabled             bool          `json:"disabled"`
	Notifications        []interface{} `json:"notifications"`
	ParameterizedBody    string        `json:"parameterizedBody,omitempty"`
	ParameterizedSubject string        `json:"parameterizedSubject,omitempty"`
	Severity             string        `json:"severity"`
}

type VisualizationOptions struct {
	DisableSampling     bool                 `json:"disableSampling"`
	PublishLabelOptions []PublishLabelOption `json:"publishLabelOptions"`
	ShowDataMarkers     bool                 `json:"showDataMarkers"`
	ShowEventLines      bool                 `json:"showEventLines"`
	Time                TimeOptions          `json:"time"`
}

type PublishLabelOption struct {
	DisplayName  string  `json:"displayName"`
	Label        string  `json:"label"`
	PaletteIndex *int    `json:"paletteIndex"` // pointer for null values
	ValuePrefix  *string `json:"valuePrefix"`
	ValueSuffix  *string `json:"valueSuffix"`
	ValueUnit    *string `json:"valueUnit"`
}

type TimeOptions struct {
	Range    int64  `json:"range"`
	RangeEnd int64  `json:"rangeEnd"`
	Type     string `json:"type"`
}

type Incident struct {
	Active                      bool    `json:"active"`
	AnomalyState                string  `json:"anomalyState"`
	AnomalyStateUpdateTimestamp int64   `json:"anomalyStateUpdateTimestamp"`
	DetectLabel                 string  `json:"detectLabel"`
	DetectorId                  string  `json:"detectorId"`
	DetectorName                string  `json:"detectorName"`
	DisplayBody                 string  `json:"displayBody"`
	Events                      []Event `json:"events"`
	IncidentId                  string  `json:"incidentId"`
	IsMuted                     bool    `json:"isMuted"`
	Severity                    string  `json:"severity"`
	TriggeredNotificationSent   bool    `json:"triggeredNotificationSent"`
	TriggeredWhileMuted         bool    `json:"triggeredWhileMuted"`
}

type Event struct {
	AnomalyState     string                 `json:"anomalyState"`
	DetectLabel      string                 `json:"detectLabel"`
	DetectorId       string                 `json:"detectorId"`
	DetectorName     string                 `json:"detectorName"`
	EventAnnotations map[string]interface{} `json:"event_annotations"`
	ID               string                 `json:"id"`
	IncidentId       string                 `json:"incidentId"`
	Inputs           map[string]Input       `json:"inputs"`
	LinkedTeams      []string               `json:"linkedTeams"`
	Severity         string                 `json:"severity"`
	Timestamp        int64                  `json:"timestamp"`
}

type Input struct {
	Key   map[string]string `json:"key,omitempty"`
	Value string            `json:"value"`
}
