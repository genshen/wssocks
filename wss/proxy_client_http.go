package wss

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"net/url"
)

type HttpClient struct {
	wsc *WebSocketClient
	//log *term_view.ProgressLog
}

func NewHttpProxy(wsc *WebSocketClient) HttpClient {
	return HttpClient{wsc: wsc}
}

func (client *HttpClient) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// todo filter header.
	closed := make(chan bool)
	cherr := make(chan error)
	server := make(chan ServerData)
	defer close(closed)
	defer close(cherr)
	defer close(server)

	proxy := client.wsc.NewProxy(server, closed, cherr)
	defer client.wsc.RemoveProxy(proxy.Id)
	defer client.wsc.TellClose(proxy.Id) // todo

	// establish with header fixme plog
	var headerBuffer bytes.Buffer
	headerBuffer.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.String(), req.Proto))
	for k, v := range req.Header {
		headerBuffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	headerBuffer.WriteString("\r\n")

	if err := proxy.Establish(nil, client.wsc, headerBuffer.Bytes(), ProxyTypeHttp,
		net.JoinHostPort(req.Host, "80")); err != nil { // fixme default port
		log.Error("write header error:", err)
		return
	}
	// copy data
	writer := WebSocketWriter{WSC: &client.wsc.ConcurrentWebSocket, Id: proxy.Id}
	if _, err := io.Copy(&writer, req.Body); err != nil {
		log.Error("write body error:", err)
	}

	// read from server and write server data to client.
	for {
		select {
		case err := <-cherr: // errors receiving from server.
			log.Error(err)
			return
		case tellClose := <-closed:
			if tellClose {
				if err := client.wsc.TellClose(proxy.Id); err != nil {
					log.Error(err)
				}
			}
			return
		case ser := <-server: // fixme can be multiple parts
			log.Println("hi0")
			reader := bufio.NewReader(bytes.NewBuffer(ser.Data))
			if newResp, err := http.ReadResponse(reader, req); err != nil {
				log.Error(err)
				return
			} else {
				copyHeaders(w.Header(), newResp.Header)
				w.WriteHeader(newResp.StatusCode)
				if _, err := io.Copy(w, newResp.Body); err != nil {
					log.Error(err)
				}
				if err := newResp.Body.Close(); err != nil {
					log.Error(err)
				}
			}
			return
		}
	}
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
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
