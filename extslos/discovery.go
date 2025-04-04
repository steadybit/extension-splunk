/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extslos

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
	TargetType                = "com.steadybit.extension_splunk.slo"
	targetIcon                = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSI+PHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMiAyQzExLjE2MTQgMiAxMC40NDMzIDIuNTE2MTYgMTAuMTQ2MSAzLjI0ODEyQzcuMTc5ODMgNC4wNjA3MiA1IDYuNzc1NzkgNSAxMFYxNC42OTcyTDMuMTY3OTUgMTcuNDQ1M0MyLjk2MzM4IDE3Ljc1MjIgMi45NDQzMSAxOC4xNDY3IDMuMTE4MzMgMTguNDcxOUMzLjI5MjM1IDE4Ljc5NyAzLjYzMTIxIDE5IDQgMTlIOC41MzU0NEM4Ljc3ODA2IDIwLjY5NjEgMTAuMjM2OCAyMiAxMiAyMkMxMy43NjMyIDIyIDE1LjIyMTkgMjAuNjk2MSAxNS40NjQ2IDE5SDIwQzIwLjM2ODggMTkgMjAuNzA3NyAxOC43OTcgMjAuODgxNyAxOC40NzE5QzIxLjA1NTcgMTguMTQ2NyAyMS4wMzY2IDE3Ljc1MjIgMjAuODMyIDE3LjQ0NTNMMTkgMTQuNjk3MlYxMEMxOSA2Ljc3NTc5IDE2LjgyMDIgNC4wNjA3MiAxMy44NTM5IDMuMjQ4MTJDMTMuNTU2NyAyLjUxNjE2IDEyLjgzODYgMiAxMiAyWk0xMiAyMEMxMS4zNDY5IDIwIDEwLjc5MTMgMTkuNTgyNiAxMC41ODU0IDE5SDEzLjQxNDZDMTMuMjA4NyAxOS41ODI2IDEyLjY1MzEgMjAgMTIgMjBaTTE3IDEyLjQ1NDlWMTAuNTg0Nkw4IDZWOC4wNTI5NEwxNC45NzU0IDExLjVMOCAxNC45OTE1VjE3TDE3IDEyLjQ1OThWMTIuNDU0OVoiIGZpbGw9ImN1cnJlbnRDb2xvciIvPjwvc3ZnPg=="
	stateCheckModeAtLeastOnce = "atLeastOnce"
	stateCheckModeAllTheTime  = "allTheTime"
	attributeName             = "splunk.slo.name"
	attributeID               = "splunk.slo.id"
	attributeIndicator        = "splunk.slo.indicator"
	attributeCreator          = "splunk.slo.creator"
)

type sloDiscovery struct {
}

var (
	_           discovery_kit_sdk.TargetDescriber    = (*sloDiscovery)(nil)
	_           discovery_kit_sdk.AttributeDescriber = (*sloDiscovery)(nil)
	RestyClient *resty.Client
)

func NewSLODiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &sloDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *sloDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: TargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *sloDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       TargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Splunk SLO", Other: "Splunk SLOs"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(targetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: attributeName},
				{Attribute: attributeIndicator},
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

func (d *sloDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
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
				One:   "name",
				Other: "names",
			},
		}, {
			Attribute: attributeIndicator,
			Label: discovery_kit_api.PluralLabel{
				One:   "Indicator",
				Other: "Indicators",
			},
		}, {
			Attribute: attributeCreator,
			Label: discovery_kit_api.PluralLabel{
				One:   "Creator",
				Other: "Creators",
			},
		},
	}
}

func (d *sloDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllSLOs(ctx, RestyClient), nil
}

func getAllSLOs(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 1000)

	var splunkResponse Response
	res, err := client.R().
		SetContext(ctx).
		SetResult(&splunkResponse).
		SetBody(`{}`).
		Post("/v2/slo/search")

	if err != nil {
		log.Err(err).Msgf("Failed to retrieve detectors from Splunk. Full response: %v", res.String())
		return result
	}

	if res.StatusCode() != 200 && res.StatusCode() != 404 {
		log.Warn().Msgf("Splunk API responded with unexpected status code %d while retrieving slos. Full response: %v",
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
					attributeID:        {detector.ID},
					attributeName:      {detector.Name},
					attributeIndicator: {detector.Indicator},
					attributeCreator:   {detector.Creator},
				}})
		}
	}

	return discovery_kit_commons.ApplyAttributeExcludes(result, config.Config.DiscoveryAttributesExcludesSLO)
}
