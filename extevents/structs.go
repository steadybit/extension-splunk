package extevents

type Event struct {
	Category   string            `json:"category"`
	EventType  string            `json:"eventType"`
	Dimensions map[string]string `json:"dimensions"`
	Properties map[string]string `json:"properties"`
	Timestamp  int64             `json:"timestamp"`
}
