package caddyhandlerPlugin

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"go.uber.org/zap"

	client "github.com/gsmlg-dev/caddy-handler-plugin/client"
)

const DirectiveName = "handler_plugin"

func init() {
	httpcaddyfile.RegisterHandlerDirective(DirectiveName, parseCaddyfile)

	caddy.RegisterModule(CaddyHandlerPlugin{})
}

type CaddyHandlerPlugin struct {
	PluginPath string `json:"plugin_path,omitempty"`
	client     *client.HandlerClient

	logger *zap.Logger
}

func (CaddyHandlerPlugin) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers." + DirectiveName,
		New: func() caddy.Module { return new(CaddyHandlerPlugin) },
	}
}

func (chp *CaddyHandlerPlugin) Provision(ctx caddy.Context) error {
	chp.logger = ctx.Logger(chp)

	return chp.loadPlugin()
}

func (chp *CaddyHandlerPlugin) Cleanup() error {
	return chp.unloadPlugin()
}

func (chp *CaddyHandlerPlugin) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return chp.client.Serve(w, r, next)
}

func (chp *CaddyHandlerPlugin) loadPlugin() error {
	if chp.PluginPath == "" {
		chp.logger.Error("plugin_path is not set, cannot load plugin")
		return fmt.Errorf("plugin_path is required")
	} else {
		c, err := client.New(chp.PluginPath)
		if err != nil {
			return err
		}
		chp.client = c

		return nil
	}
}

func (chp *CaddyHandlerPlugin) unloadPlugin() error {
	chp.client.Kill()
	return nil
}

// parseCaddyfile parses the handler_plugin directive.
//
//	handler_plugin {
//	  plugin_path   <path>
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var chp CaddyHandlerPlugin

	for h.Next() {
		for h.NextBlock(0) {
			switch h.Val() {
			case "plugin_path":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				chp.PluginPath = h.Val()

			default:
				return nil, h.Errf("unknown subdirective '%s'", h.Val())
			}
		}
	}

	return &chp, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*CaddyHandlerPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*CaddyHandlerPlugin)(nil)
)
