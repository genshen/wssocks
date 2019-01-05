package ws_socks

const (
	WebSocketMessageTypeProxy     = "proxy"
	WebSocketMessageTypeHeartbeat = "heartbeat"
	WebSocketMessageTypeRequest   = "request"
)

const (
	WsTpClose = "finish"
	WsTpData  = "data"
	WsTpEst   = "est" // establish
)

type WebSocketMessage2 struct {
	Id   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"` // json.RawMessage
}

// Proxy data (from server to client)
type ProxyData struct {
	DataBase64 string `json:"base64"`
}

type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"` // json.RawMessage
}

// Proxy message
type ProxyMessage struct {
	Addr string `json:"addr"`
}

// request message (from client to server)
type RequestMessage ProxyData
