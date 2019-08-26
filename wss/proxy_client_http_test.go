package wss

import (
	"fmt"
	"testing"
)

const TestHttpHeader = `GET http://golang.org/abc?d=gh#hash HTTP/1.1
Host: golang.org
Connection: keep-alive
Proxy-Connection: keep-alive
Accept: */*
`

func TestHttpHeaderPath(t *testing.T) {
	var client HttpClient

	var expectedNewHeader = `GET /abc?d=gh#hash HTTP/1.1
Host: golang.org
Connection: keep-alive
Proxy-Connection: keep-alive
Accept: */*
`

	var buffer = []byte(TestHttpHeader)
	if newBuffer, err := client.EstablishData(buffer, ); err != nil {
		t.Fail()
	} else {
		if string(newBuffer) != expectedNewHeader {
			fmt.Printf("new buffer is: \n%s", string(newBuffer))
			t.Fail()
		}
	}
}

func TestHttpHeaderParser(t *testing.T) {
	var client HttpClient
	var buffer = []byte(TestHttpHeader)
	if addrHttp, err := client.ParseHeader(nil, buffer, ); err != nil {
		t.Fail()
	} else {
		if addrHttp != "golang.org:80" {
			fmt.Println("addr is: ", addrHttp)
			t.Fail()
		}
	}
}
