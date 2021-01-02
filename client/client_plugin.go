package client

import (
	"errors"
	"github.com/genshen/wssocks/wss"
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
	VersionPlugin VersionPlugin
}

var ErrPluginOccupied = errors.New("the plugin is occupied by another plugin")

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
func AddPluginRedirect(redirect RequestPlugin) error {
	if clientPlugin.RequestPlugin != nil {
		return ErrPluginOccupied
	}
	clientPlugin.RequestPlugin = redirect
	return nil
}

func AddPluginVersion(verPlugin VersionPlugin) error {
	if clientPlugin.VersionPlugin != nil {
		return ErrPluginOccupied
	}
	clientPlugin.VersionPlugin = verPlugin
	return nil
}
