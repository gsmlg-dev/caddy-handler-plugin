package client

import (
	"net/http"
	"os/exec"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
	"github.com/hashicorp/go-plugin"
)

type HandlerClient struct {
	client  *plugin.Client
	handler *shared.HandlerRPC
}

func (c *HandlerClient) Kill() {
	c.client.Kill()
}

func (c *HandlerClient) SetConfig(cfg map[string][]string) (bool, error) {
	return c.handler.SetConfig(cfg)
}

func (c *HandlerClient) Serve(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	q := shared.CreatePluginQuery(r)

	reply, err := c.handler.Serve(q)
	if err != nil {
		return err
	}

	return reply.Serve(w, r, next)
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
		client:  client,
		handler: h.(*shared.HandlerRPC),
	}, err
}
