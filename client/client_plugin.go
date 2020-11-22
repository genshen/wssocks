package client

import (
	"github.com/genshen/wssocks/wss"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

type RequestPlugin interface {
	// in the plugin, we may add http header and modify remote address.
	BeforeRequest(hc *http.Client, transport *http.Transport, url *url.URL, header *http.Header) error
}

type VersionPlugin interface {
	OnServerVersion(ver wss.VersionNeg) error
}

type Plugin struct {
	RequestPlugin RequestPlugin
	VersionPlugin  VersionPlugin
}

// check whether the request plugin has been added.
// this plugin can only be at most one instance.
func (plugin *Plugin) HasRequestPlugin() bool {
	return plugin.RequestPlugin != nil
}

func (plugin *Plugin) HasVersionPlugin() bool {
	return plugin.VersionPlugin != nil
}

var clientPlugin Plugin

// add a client plugin
func AddPluginRedirect(redirect RequestPlugin) {
	if clientPlugin.RequestPlugin != nil {
		log.Fatal("redirect plugin has been occupied by another plugin.")
		return
	}
	clientPlugin.RequestPlugin = redirect
}

func AddPluginVersion(verPlugin VersionPlugin) {
	if clientPlugin.VersionPlugin != nil {
		log.Fatal("version plugin has been occupied by another plugin.")
		return
	}
	clientPlugin.VersionPlugin = verPlugin
}
