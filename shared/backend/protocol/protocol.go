package protocol

type Message struct {
	UserID    string `json:"user_id"`
	RoomID    string `json:"room_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	NodeID        string `json:"node_id,omitempty"`
	Connections   int    `json:"connections,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	SourceService string `json:"source_service,omitempty"`
}

type HealthResponse struct {
	Status string `json:"status"`
	NodeID string `json:"node_id"`
}
