package wss

import (
	"fmt"
	"testing"
)

func TestHttpHeaderParse(t *testing.T) {
	var client Client
	var header = `GET http://golang.org/abc?d=gh#hash HTTP/1.1
Host: golang.org
Connection: keep-alive
Proxy-Connection: keep-alive
Accept: */*
`
	var expectedNewHeader = `GET /abc?d=gh#hash HTTP/1.1
Host: golang.org
Connection: keep-alive
Proxy-Connection: keep-alive
Accept: */*
`

	var buffer = []byte(header)
	if addrHttp, newBuffer, err := client.parseHttpHeader(buffer, len(buffer)); err != nil {
		t.Fail()
	} else {
		if addrHttp != "golang.org:80" {
			fmt.Println("addr is: ", addrHttp)
			t.Fail()
		}
		if string(newBuffer) != expectedNewHeader {
			fmt.Printf("new buffer is: \n%s", string(newBuffer))
			t.Fail()
		}
	}
}
