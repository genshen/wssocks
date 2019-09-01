package wss

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

func HttpRequestHeader(buffer *bytes.Buffer, req *http.Request) {
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.String(), req.Proto))
	//	req.Header.Add("Connection", "close")
	for name, headers := range req.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			buffer.WriteString(fmt.Sprintf("%v: %v\r\n", name, h))
		}
	}
	buffer.WriteString("\r\n")
}

func HttpRespHeader(buffer *bytes.Buffer, resp *http.Response) {
	buffer.Write([]byte(fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status)))
	//req.Header.Add("Connection", "close")
	for name, headers := range resp.Header {
		for _, h := range headers {
			buffer.Write([]byte(fmt.Sprintf("%s: %s\r\n", name, h)))
		}
	}
	buffer.Write([]byte("\r\n"))
}
