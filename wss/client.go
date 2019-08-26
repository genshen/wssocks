package wss

import (
	"net"
)

const (
	ProxyTypeSocks5 = iota
	ProxyTypeHttp
	ProxyTypeHttps
)

type ProxyInterface interface {
	ProxyType() int
	// return a bool value to indicate whether it is the matched protocol.
	Trigger(data []byte) bool
	// parse protocol header bytes, return target host.
	ParseHeader(conn net.Conn, header []byte) (string, error)
	// return data transformed in connection establishing step.
	EstablishData(origin []byte) ([]byte, error)
}
