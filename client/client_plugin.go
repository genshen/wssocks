package client

import (
	"errors"
	"github.com/genshen/wssocks/wss"
	"net/http"
	"net/url"
)

// pass read-only connection option to OnOptionSet when options are set.
// we can check connection options by returning an error and may set RemoteUrl here.
type OptionPlugin interface {
	OnOptionSet(options Options) error
}

// in the plugin, we may add http header and modify remote address.
type RequestPlugin interface {
	BeforeRequest(hc *http.Client, transport *http.Transport, url *url.URL, header *http.Header) error
}

type VersionPlugin interface {
	OnServerVersion(ver wss.VersionNeg) error
}

// Plugins is a collection of all possible plugins on client
type Plugin struct {
	OptionPlugin  OptionPlugin
	RequestPlugin RequestPlugin
	VersionPlugin VersionPlugin
}

var ErrPluginOccupied = errors.New("the plugin is occupied by another plugin")

// check whether the option plugin has been added.
// this plugin can only be at most one instance.
func (plugin *Plugin) HasOptionPlugin() bool {
	return plugin.OptionPlugin != nil
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
// add an option plugin
func AddPluginOption(opPlugin OptionPlugin) error {
	if clientPlugin.OptionPlugin != nil {
		return ErrPluginOccupied
	}
	clientPlugin.OptionPlugin = opPlugin
	return nil
}

// add a client plugin
func AddPluginRequest(redirect RequestPlugin) error {
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
