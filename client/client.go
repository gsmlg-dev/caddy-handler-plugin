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

func (c *HandlerClient) Serve(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	q := shared.PluginQuery{
		// Config: r.URL.Query(),
		Path:   r.URL.Path,
		Header: r.Header,
	}

	reply, err := c.handler.Serve(q)
	if err != nil {
		return err
	}

	if reply.Done {
		if reply.Status > 0 {
			w.WriteHeader(reply.Status)
		}
		for k, v := range reply.Header {
			for i, v := range v {
				if i == 0 {
					w.Header().Set(k, v)
				} else {
					w.Header().Add(k, v)
				}
			}
		}
		w.Write(reply.Body)
		return nil
	}
	return next.ServeHTTP(w, r)
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
