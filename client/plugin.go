package client

import (
	"net/http"
	"net/url"
)

type plugin interface {
	BeforeRequest(url *url.URL, header http.Header)
}

var clientPlugin plugin = nil
