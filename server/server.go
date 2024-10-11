package server

import (
	"net/http"

	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
	"github.com/hashicorp/go-plugin"
)

type HandlerServer struct {
}

func (g *HandlerServer) Serve(r http.Request, reply *shared.PluginReply) error {
	reply.Done = false
	reply.Body = []byte("Hello!")
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
