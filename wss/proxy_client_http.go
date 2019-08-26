package wss

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/url"
)

type HttpClient struct {
}

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

func (client *HttpClient) ProxyType() int {
	return ProxyTypeHttp
}

func (client *HttpClient) Trigger(data []byte) bool {
	// now, we only support GET and POST request.
	return (len(data) > len("GET") && string(data[:len("GET")]) == "GET") ||
		(len(data) > len("POST") && string(data[:len("POST")]) == "POST")
}

func (client *HttpClient) EstablishData(origin []byte) ([]byte, error) {
	if method, address, ver, n, err := client.parseFirstLine(origin); err != nil {
		return nil, err
	} else {
		if u, err := url.Parse(address); err != nil {
			return nil, err
		} else {
			// get path?query#fragment
			u.Host = ""
			u.Scheme = ""
			newBuff := bytes.NewBuffer(nil)
			newBuff.WriteString(fmt.Sprintf("%s %s %s", method, u.String(), ver))
			newBuff.Write(origin[n:]) // append origin header and body data.
			return newBuff.Bytes(), nil
		}
	}
}

// parsing http header, and return address and parsing error
func (client *HttpClient) ParseHeader(conn net.Conn, header []byte) (string, error) {
	if _, address, _, _, err := client.parseFirstLine(header); err != nil {
		return "", err
	} else {
		if u, err := url.Parse(address); err != nil {
			return "", err
		} else {
			var host string
			// parsing port and host
			if u.Opaque == "80" { // https
				host = u.Scheme + ":80"
			} else { // http
				if u.Port() == "" {
					host = net.JoinHostPort(u.Host, "80")
				} else {
					host = net.JoinHostPort(u.Host, u.Port())
				}

			}
			return host, nil
		}
	}
}

// parse first line of http header, returning method, address, http version and the bytes of first line.
func (client *HttpClient) parseFirstLine(data []byte) (method, address, ver string, n int, err error) {
	buff := bytes.NewBuffer(data)
	if line, _, err := bufio.NewReader(buff).ReadLine(); err != nil {
		return "", "", "", len(line), err
	} else {
		var method, address, ver string
		if _, err := fmt.Sscanf(string(line), "%s %s %s", &method, &address, &ver); err != nil {
			return "", "", "", len(line), err
		} else {
			return method, address, ver, len(line), nil
		}
	}
}
