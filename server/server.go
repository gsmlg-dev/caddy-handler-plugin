// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"net/http"

	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
	"github.com/hashicorp/go-plugin"
)

type HandlerServer struct {
}

func (g *HandlerServer) Serve(r http.Request, reply *shared.PluginReply) error {
	reply.Done = false
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
