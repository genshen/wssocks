package wss

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
)

const (
	ProxyTypeSocks5 = iota
	ProxyTypeHttp
	ProxyTypeHttps
)

// client part of socks5 server
type Client struct {
}

// response to socks5 client and start to exchange data between socks5 client and
func (client *Client) Reply(conn net.Conn, onDial func(conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error) error {
	defer conn.Close()
	var buffer [1024]byte
	var firstSendData []byte = nil
	var addr string
	var proxyType int

	n, err := conn.Read(buffer[:])
	if err != nil {
		return err
	}

	if n >= 2 && buffer[0] == 0x05 { //sock5 proxy
		if addrSocks5, err := client.parseSocks5Header(conn); err != nil {
			return err
		} else {
			proxyType = ProxyTypeSocks5
			addr = addrSocks5
		}
	} else if n > len("CONNECT") && string(buffer[:len("CONNECT")]) == "CONNECT" {
		// is https proxy
		if addrHttp, err := client.parseHttpsHeader(buffer[:], n); err != nil {
			return err
		} else {
			proxyType = ProxyTypeHttps
			addr = addrHttp
		}
	} else if (n > len("GET") && string(buffer[:len("GET")]) == "GET") ||
		(n > len("POST") && string(buffer[:len("POST")]) == "POST") {
		// is http proxy
		if addrHttp, newBuffer, err := client.parseHttpHeader(buffer[:n], n); err != nil {
			return err
		} else {
			proxyType = ProxyTypeHttp
			addr = addrHttp
			firstSendData = newBuffer
		}
	} else {
		return errors.New("only socks5 or http proxy")
	}

	//  dial to target.
	// firstSendData can be nil, which means there is no data to be send during connection establishing.
	if err := onDial(conn.(*net.TCPConn), firstSendData, proxyType, addr); err != nil {
		return err
	}

	return nil
}

// parsing socks5 header, and return address and parsing error
func (client *Client) parseSocks5Header(conn net.Conn) (string, error) {
	// response to socks5 client
	// see rfc 1982 for more details (https://tools.ietf.org/html/rfc1928)
	n, err := conn.Write([]byte{0x05, 0x00}) // version and no authentication required
	if err != nil {
		return "", err
	}

	// step2: process client Requests and does Reply
	/**
	+----+-----+-------+------+----------+----------+
	|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	+----+-----+-------+------+----------+----------+
	| 1  |  1  | X'00' |  1   | Variable |    2     |
	+----+-----+-------+------+----------+----------+
	*/
	var buffer [1024]byte

	n, err = conn.Read(buffer[:])
	if err != nil {
		return "", err
	}
	if n < 6 {
		return "", errors.New("not a socks protocol")
	}

	var host string
	switch buffer[3] {
	case 0x01:
		// ipv4 address
		ipv4 := make([]byte, 4)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[4:]), ipv4, len(ipv4)); err != nil {
			return "", err
		}
		host = net.IP(ipv4).String()
	case 0x04:
		// ipv6
		ipv6 := make([]byte, 16)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[4:]), ipv6, len(ipv6)); err != nil {
			return "", err
		}
		host = net.IP(ipv6).String()
	case 0x03:
		// domain
		addrLen := int(buffer[4])
		domain := make([]byte, addrLen)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[5:]), domain, addrLen); err != nil {
			return "", err
		}
		host = string(domain)
	}

	port := make([]byte, 2)
	err = binary.Read(bytes.NewReader(buffer[n-2:n]), binary.BigEndian, &port)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, strconv.Itoa((int(port[0])<<8)|int(port[1]))), nil
}

// parsing https header, and return address and parsing error
func (client *Client) parseHttpsHeader(buffer []byte, n int) (string, error) {
	buff := bytes.NewBuffer(buffer)
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
						host = u.Host + ":443"
					} else {
						host = u.Host + ":" + u.Port()
					}
				}
				return host, nil
			}
		}
	}
}

// parsing http header, and return address and parsing error
func (client *Client) parseHttpHeader(buffer []byte, n int) (string, []byte, error) {
	buff := bytes.NewBuffer(buffer)
	if line, _, err := bufio.NewReader(buff).ReadLine(); err != nil {
		return "", nil, err
	} else {
		var method, address, ver string
		if _, err := fmt.Sscanf(string(line), "%s %s %s", &method, &address, &ver); err != nil {
			return "", nil, err
		} else {
			if u, err := url.Parse(address); err != nil {
				return "", nil, err
			} else {
				var host string
				// parsing port and host
				if u.Opaque == "80" { // https
					host = u.Scheme + ":80"
				} else { // http
					if u.Port() == "" {
						host = u.Host + ":80"
					} else {
						host = u.Host + ":" + u.Port()
					}

				}
				// get path?query#fragment
				u.Host = ""
				u.Scheme = ""
				newBuff := bytes.NewBuffer(nil)
				newBuff.WriteString(fmt.Sprintf("%s %s %s", method, u.String(), ver))
				newBuff.Write(buffer[len(line):]) // append origin header and body data.
				return host, newBuff.Bytes(), nil
			}
		}
	}
}
