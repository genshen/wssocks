package wss

import (
	"net"
)

const (
	ProxyTypeSocks5 = iota
	ProxyTypeHttp
	ProxyTypeHttps
)

func ProxyTypeStr(tp int) string {
	switch tp {
	case ProxyTypeHttp:
		return "http"
	case ProxyTypeHttps:
		return "https"
	case ProxyTypeSocks5:
		return "socks5"
	}
	return "unknown"
}

// interface of proxy client, supported types: http/https/socks5
type ProxyInterface interface {
	ProxyType() int
	// return a bool value to indicate whether it is the matched protocol.
	Trigger(data []byte) bool
	// parse protocol header bytes, return target host.
	ParseHeader(conn net.Conn, header []byte) (string, error)
	// return data transformed in connection establishing step.
	EstablishData(origin []byte) ([]byte, error)
}
