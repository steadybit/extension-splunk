/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdetectors

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-splunk/config"
	"time"
)

const (
	TargetType                = "com.steadybit.extension_splunk.detector"
	targetIcon                = "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPGcgaWQ9Ikljb24gLyBBbGVydHMiPgo8cGF0aCBpZD0iRGVmYXVsdCIgZD0iTTQuOTk5OTYgOS41QzQuOTk5OTYgNS42MzQwMSA4LjEzMzk2IDIuNSAxMiAyLjVDMTUuODY1OSAyLjUgMTkgNS42MzQwMSAxOSA5LjVWMTIuNUwyMC4yNzYzIDE1LjA1MjhDMjAuNjA4OCAxNS43MTc3IDIwLjEyNTMgMTYuNSAxOS4zODE5IDE2LjVINC42MTc5OUMzLjg3NDYxIDE2LjUgMy4zOTExMSAxNS43MTc3IDMuNzIzNTYgMTUuMDUyOEw0Ljk5OTk2IDEyLjVWOS41WiIvPgo8cGF0aCBpZD0iV2VhayIgZD0iTTkgMTguNUgxNUMxNSAyMC4xNTY5IDEzLjY1NjkgMjEuNSAxMiAyMS41QzEwLjM0MzEgMjEuNSA5IDIwLjE1NjkgOSAxOC41WiIvPgo8L2c+Cjwvc3ZnPg=="
	stateCheckModeAtLeastOnce = "atLeastOnce"
	stateCheckModeAllTheTime  = "allTheTime"
	attributeName             = "splunk.detector.name"
	attributeID               = "splunk.detector.id"
	attributeDescription      = "splunk.detector.description"
	attributeStatus           = "splunk.detector.status"
	attributeCreator          = "splunk.detector.creator"
	attributeDetectorOrigin   = "splunk.detector.detectorOrigin"
)

type alertDiscovery struct {
}

var (
	_           discovery_kit_sdk.TargetDescriber    = (*alertDiscovery)(nil)
	_           discovery_kit_sdk.AttributeDescriber = (*alertDiscovery)(nil)
	RestyClient *resty.Client
)

func NewDetectorDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &alertDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *alertDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: TargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *alertDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       TargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Splunk detector", Other: "Splunk detectors"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(targetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: attributeID},
				{Attribute: attributeName},
				{Attribute: attributeDescription},
				{Attribute: attributeStatus},
				{Attribute: attributeCreator},
				{Attribute: attributeDetectorOrigin},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: attributeName,
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *alertDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: attributeID,
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: attributeName,
			Label: discovery_kit_api.PluralLabel{
				One:   "Detector name",
				Other: "Detector names",
			},
		}, {
			Attribute: attributeDescription,
			Label: discovery_kit_api.PluralLabel{
				One:   "Description",
				Other: "Descriptions",
			},
		}, {
			Attribute: attributeStatus,
			Label: discovery_kit_api.PluralLabel{
				One:   "Status",
				Other: "Status",
			},
		}, {
			Attribute: attributeCreator,
			Label: discovery_kit_api.PluralLabel{
				One:   "Creator",
				Other: "Creators",
			},
		}, {
			Attribute: attributeDetectorOrigin,
			Label: discovery_kit_api.PluralLabel{
				One:   "Origin",
				Other: "Origins",
			},
		},
	}
}

func (d *alertDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllDetectors(ctx, RestyClient), nil
}

func getAllDetectors(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 1000)

	var splunkResponse Response
	res, err := client.R().
		SetContext(ctx).
		SetResult(&splunkResponse).
		Get("/v2/detector")

	if err != nil {
		log.Err(err).Msgf("Failed to retrieve detectors from Splunk. Full response: %v", res.String())
		return result
	}

	if res.StatusCode() != 200 && res.StatusCode() != 404 {
		log.Warn().Msgf("Splunk API responded with unexpected status code %d while retrieving detectors. Full response: %v",
			res.StatusCode(),
			res.String())
		return result
	} else {
		log.Trace().Msgf("Splunk response: %v", splunkResponse)

		for _, detector := range splunkResponse.Results {
			result = append(result, discovery_kit_api.Target{
				Id:         detector.ID,
				TargetType: TargetType,
				Label:      detector.Name,
				Attributes: map[string][]string{
					attributeID:             {detector.ID},
					attributeName:           {detector.Name},
					attributeDescription:    {detector.Description},
					attributeStatus:         {detector.Status},
					attributeCreator:        {detector.Creator},
					attributeDetectorOrigin: {detector.DetectorOrigin},
				}})
		}
	}

	return discovery_kit_commons.ApplyAttributeExcludes(result, config.Config.DiscoveryAttributesExcludesAlert)
}
