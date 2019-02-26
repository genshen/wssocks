package ws_socks

const (
	WsTpVer  = "version"
	WsTpClose = "finish"
	WsTpData  = "data"
	WsTpEst   = "est" // establish
)

type WebSocketMessage struct {
	Id   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"` // json.RawMessage
}

// Proxy data (from server to client)
type ProxyData struct {
	DataBase64 string `json:"base64"`
}

// Proxy message
type ProxyMessage struct {
	Addr string `json:"addr"`
}

// request message (from client to server)
type RequestMessage ProxyData
