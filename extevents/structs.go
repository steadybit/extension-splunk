package extevents

type Event struct {
	Category   string            `json:"category"`
	EventType  string            `json:"eventType"`
	Properties map[string]string `json:"properties"`
	Timestamp  int64             `json:"timestamp"`
}
