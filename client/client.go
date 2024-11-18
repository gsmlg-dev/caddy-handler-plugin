package client

import (
	"net/http"
	"os/exec"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"

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

struct pluginLogger {
	logger *zap.Logger
}

func (pl *pluginLogger) Write(p []byte) (n int, err error) {
	msg := string(p[:])
	pl.logger.Log(zap.DebugLevel, msg)
	len(p), nil
}

func New(path string, zapLogger *zap.Logger) (*HandlerClient, error) {
	writer = pluginLogger{logger: zapLogger}
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "handler-plugin",
		Output: writer,
		Level:  hclog.Debug,
	})

	var pluginMap = map[string]plugin.Plugin{
		"handler": &shared.HandlerPlugin{},
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(path),
		Logger:          logger,
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
