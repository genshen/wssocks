package client

import (
	"github.com/genshen/wssocks/wss"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

type ServerRedirect interface {
	// in the plugin, we may add http header and modify remote address.
	BeforeRequest(dialer *websocket.Dialer, url *url.URL, header http.Header) error
}

type VersionPlugin interface {
	OnServerVersion(ver wss.VersionNeg) error
}

type Plugin struct {
	RedirectPlugin ServerRedirect
	VersionPlugin  VersionPlugin
}

// check whether the redirect plugin has been added.
// this plugin can only be at most one instance.
func (plugin *Plugin) HasRedirectPlugin() bool {
	return plugin.RedirectPlugin != nil
}

func (plugin *Plugin) HasVersionPlugin() bool {
	return plugin.VersionPlugin != nil
}

var clientPlugin Plugin

// add a client plugin
func AddPluginRedirect(redirect ServerRedirect) {
	if clientPlugin.RedirectPlugin != nil {
		log.Fatal("redirect plugin has been occupied by another plugin.")
		return
	}
	clientPlugin.RedirectPlugin = redirect
}

func AddPluginVersion(verPlugin VersionPlugin) {
	if clientPlugin.VersionPlugin != nil {
		log.Fatal("version plugin has been occupied by another plugin.")
		return
	}
	clientPlugin.VersionPlugin = verPlugin
}
