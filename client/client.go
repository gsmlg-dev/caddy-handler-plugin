package client

import (
	"net/http"
	"os/exec"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
	"github.com/hashicorp/go-plugin"
)

type HandlerClient struct {
	client    *plugin.Client
	rpcClient plugin.ClientProtocol
	handler   shared.HandlerRPC
}

func (c *HandlerClient) Kill() {
	c.client.Kill()
}

func (c *HandlerClient) Serve(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return c.handler.Serve(w, r, next)
}

func New(path string) (*HandlerClient, error) {

	var pluginMap = map[string]plugin.Plugin{
		"handler": &shared.HandlerPlugin{},
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(path),
	})

	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	h, err := rpcClient.Dispense("handler")
	if err != nil {
		return nil, err
	}

	return &HandlerClient{
		client:    client,
		rpcClient: rpcClient,
		handler:   h.(shared.HandlerRPC),
	}, err
}
