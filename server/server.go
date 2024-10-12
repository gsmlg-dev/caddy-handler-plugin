package server

import (
	"net/http"

	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
	"github.com/hashicorp/go-plugin"
)

type HandlerServerDefault struct {
	Config map[string][]string
}

func (g *HandlerServerDefault) SetConfig(cfg map[string][]string, ok *bool) error {
	g.Config = cfg
	*ok = true
	return nil
}

func (g *HandlerServerDefault) Serve(q shared.PluginQuery, reply *shared.PluginReply) error {
	reply.Done = true
	header := http.Header{}
	header.Set("Content-Type", "text/plain")
	reply.Header = header
	reply.Body = []byte("Hello World")
	return nil
}

func New(c shared.Handler) {
	handler := c

	var pluginMap = map[string]plugin.Plugin{
		"handler": &shared.HandlerPlugin{Impl: handler},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
