package wss

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/genshen/wssocks/wss/term_view"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"net/url"
)

type HttpClient struct {
	wsc  *WebSocketClient
	plog *term_view.ProgressLog
}

func NewHttpProxy(wsc *WebSocketClient, plog *term_view.ProgressLog) HttpClient {
	return HttpClient{wsc: wsc, plog: plog}
}

func (client *HttpClient) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	type Done struct {
		tell bool
		err  error
	}
	done := make(chan Done, 2)
	continued := make(chan int)
	defer close(continued)

	hj, _ := w.(http.Hijacker)
	conn, jack, _ := hj.Hijack()
	defer conn.Close()
	defer jack.Flush()

	proxy := client.wsc.NewProxy(nil, nil, nil)
	proxy.onData = func(id ksuid.KSUID, data ServerData) {
		if data.Tag == TagEstOk || data.Tag == TagEstErr {
			continued <- data.Tag
			return
		}
		if _, err := jack.Write(data.Data); err != nil {
			done <- Done{true, err}
		}
	}
	proxy.onClosed = func(id ksuid.KSUID, tell bool) {
		done <- Done{tell, nil}
	}
	proxy.onError = func(ksuids ksuid.KSUID, err error) {
		done <- Done{true, err}
	}

	// establish with header fixme plog
	if !req.URL.IsAbs() {
		client.wsc.RemoveProxy(proxy.Id)
		w.WriteHeader(403)
		_, _ = w.Write([]byte("This is a proxy server. Does not respond to non-proxy requests."))
		return
	}

	client.plog.Update(term_view.Status{IsNew: true, Address: req.URL.Host, Type: ProxyTypeHttp})
	defer client.plog.Update(term_view.Status{IsNew: false, Address: req.URL.Host, Type: ProxyTypeHttp})

	var headerBuffer bytes.Buffer
	host, _ := client.parseUrl(req.Method, req.Proto, req.URL)
	HttpRequestHeader(&headerBuffer, req)

	if err := proxy.Establish(client.wsc, headerBuffer.Bytes(), ProxyTypeHttp, host); err != nil { // fixme default port
		log.Error("write header error:", err)
		client.wsc.RemoveProxy(proxy.Id)
		if err := client.wsc.TellClose(proxy.Id); err != nil {
			log.Error("close error", err)
		}
		return
	}

	// fixme add timeout
	// wait receiving "established connection" from server
	if tag := <-continued; tag == TagEstErr {
		return
	}

	// copy request body data
	writer := WebSocketWriter{WSC: &client.wsc.ConcurrentWebSocket, Id: proxy.Id}
	if _, err := io.Copy(&writer, req.Body); err != nil {
		log.Error("write body error:", err)
		client.wsc.RemoveProxy(proxy.Id)
		if err := client.wsc.TellClose(proxy.Id); err != nil {
			log.Error("close error", err)
		}
		return
	}
	if err := client.wsc.WriteProxyMessage(proxy.Id, TagNoMore, nil); err != nil {
		log.Error("write body error:", err)
		client.wsc.RemoveProxy(proxy.Id)
		if err := client.wsc.TellClose(proxy.Id); err != nil {
			log.Error("close error", err)
		}
		return
	}

	// finished
	d := <-done // fixme add timeout
	client.wsc.RemoveProxy(proxy.Id)
	if d.tell {
		if err := client.wsc.TellClose(proxy.Id); err != nil {
			log.Error(err)
		}
	}
	if d.err != nil {
		log.Error(d.err)
	}
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func (client *HttpClient) ProxyType() int {
	return ProxyTypeHttp
}

// parse first line of http header, returning method, address, http version and the bytes of first line.
func (client *HttpClient) parseUrl(method, ver string, u *url.URL) (string, string) {
	var host string
	// parsing port and host
	if u.Opaque == "80" { // https
		host = u.Scheme + ":80"
	} else { // http
		if u.Port() == "" {
			host = net.JoinHostPort(u.Hostname(), "80")
		} else {
			host = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	// get path?query#fragment
	//u.Host = ""
	//u.Scheme = ""
	return host, u.String()
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
