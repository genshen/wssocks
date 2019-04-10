package client

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
)

type ServerRedirect interface {
	// in the plugin, we may add http header and modify remote address.
	BeforeRequest(dialer *websocket.Dialer, url *url.URL, header http.Header) error
}

type Plugin struct {
	RedirectPlugin ServerRedirect
}

// check whether the plugin has been added.
// this plugin can only be at most one instance.
func (plugin *Plugin) HasPlugin() bool {
	return plugin.RedirectPlugin != nil
}

var clientPlugin Plugin

// add a client plugin
func AddPluginRedirect(redirect ServerRedirect) {
	if clientPlugin.RedirectPlugin != nil {
		log.Fatal("this plugin has been occupied by another plugin.")
		return
	}
	clientPlugin.RedirectPlugin = redirect
}
