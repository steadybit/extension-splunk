package extslos

type Response struct {
	Count   int   `json:"count"`
	Results []Slo `json:"results"`
}

type Slo struct {
	Created       int64    `json:"created"`
	Creator       string   `json:"creator"`
	Description   *string  `json:"description"` // pointer to allow null
	ID            string   `json:"id"`
	Indicator     string   `json:"indicator"`
	Inputs        Inputs   `json:"inputs"`
	LastUpdated   int64    `json:"lastUpdated"`
	LastUpdatedBy string   `json:"lastUpdatedBy"`
	Metadata      []string `json:"metadata"`
	Name          string   `json:"name"`
	Targets       []Target `json:"targets"`
	Type          string   `json:"type"`
}

type Inputs struct {
	GoodEventsLabel  string `json:"goodEventsLabel"`
	ProgramText      string `json:"programText"`
	TotalEventsLabel string `json:"totalEventsLabel"`
}

type Target struct {
	CompliancePeriod string         `json:"compliancePeriod"`
	SLO              float64        `json:"slo"`
	SLOAlertRules    []SLOAlertRule `json:"sloAlertRules"`
	Type             string         `json:"type"`
}

type SLOAlertRule struct {
	AlertsTriggered bool   `json:"alertsTriggered"`
	Rules           []Rule `json:"rules"`
	Type            string `json:"type"`
}

type Rule struct {
	CalculatedParameters CalculatedParameters `json:"calculatedParameters"`
	DetectLabel          string               `json:"detectLabel"`
	Disabled             bool                 `json:"disabled"`
	Notifications        []string             `json:"notifications"`
	ParameterizedBody    string               `json:"parameterizedBody"`
	ParameterizedSubject string               `json:"parameterizedSubject"`
	Severity             string               `json:"severity"`
}

type CalculatedParameters struct {
	FireLasting      string  `json:"fireLasting"`
	PercentOfLasting float64 `json:"percentOfLasting"`
}

type SLOSearchConfig struct {
	BreachAlertsTriggered          bool     `json:"breachAlertsTriggered"`
	BurnRateAlertsTriggered        bool     `json:"burnRateAlertsTriggered"`
	ErrorBudgetLeftAlertsTriggered bool     `json:"errorBudgetLeftAlertsTriggered"`
	SLOIds                         []string `json:"sloIds"`
}
