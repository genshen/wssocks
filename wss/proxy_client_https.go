package wss

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/url"
)

type HttpsClient struct {
}

func (client *HttpsClient) ProxyType() int {
	return ProxyTypeHttps
}

func (client *HttpsClient) Trigger(data []byte) bool {
	return len(data) > len("CONNECT") && string(data[:len("CONNECT")]) == "CONNECT"
}

func (client *HttpsClient) EstablishData(origin []byte) ([]byte, error) {
	return nil, nil
}

// parsing https header, and return address and parsing error
func (client *HttpsClient) ParseHeader(conn net.Conn, header []byte) (string, error) {
	buff := bytes.NewBuffer(header)
	if line, _, err := bufio.NewReader(buff).ReadLine(); err != nil {
		return "", err
	} else {
		var method, address, httpVer string
		if _, err := fmt.Sscanf(string(line), "%s %s %s", &method, &address, &httpVer); err != nil {
			return "", err
		} else {
			if u, err := url.Parse(address); err != nil {
				return "", err
			} else {
				var host string
				// parsing port and host
				if u.Opaque == "443" { // https
					host = u.Scheme + ":443"
				} else { // https
					if u.Port() == "" {
						host = net.JoinHostPort(u.Host, "443")
					} else {
						host = net.JoinHostPort(u.Host, u.Port())
					}
				}
				return host, nil
			}
		}
	}
}
