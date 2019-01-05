package ws_socks

const (
	WebSocketMessageTypeProxy     = "proxy"
	WebSocketMessageTypeHeartbeat = "heartbeat"
	WebSocketMessageTypeRequest   = "request"
)

type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"` // json.RawMessage
}

// Proxy message
type ProxyMessage struct {
	Addr string `json:"addr"`
}

// request message
type RequestMessage struct {
	DataBase64 string `json:"base64"`
}
