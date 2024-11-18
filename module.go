package caddyhandlerPlugin

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"go.uber.org/zap"

	client "github.com/gsmlg-dev/caddy-handler-plugin/client"
)

const DirectiveName = "handler_plugin"

func init() {
	httpcaddyfile.RegisterHandlerDirective(DirectiveName, parseCaddyfile)
	httpcaddyfile.RegisterDirectiveOrder(DirectiveName, httpcaddyfile.After, "header")

	caddy.RegisterModule(CaddyHandlerPlugin{})
}

type CaddyHandlerPlugin struct {
	PluginPath   string              `json:"plugin_path,omitempty"`
	PluginConfig map[string][]string `json:"plugin_config,omitempty"`

	client *client.HandlerClient

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

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
//	hanler_plgun <plugin_path> {
//	    <plugin_config>
//	}
func (chp *CaddyHandlerPlugin) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	for d.Next() {
		chp.PluginPath = d.Val()
		if d.NextArg() {
			return d.Err("too many args")
		}

		cfg := make(map[string][]string)

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			name := d.Val()
			value := d.RemainingArgs()

			cfg[name] = value
		}
		chp.PluginConfig = cfg

	}
	return nil
}

func (chp *CaddyHandlerPlugin) loadPlugin() error {
	if chp.PluginPath == "" {
		chp.logger.Error("plugin_path is not set, cannot load plugin")
		return fmt.Errorf("plugin_path is required")
	} else {
		chp.logger.Info("handler-plugin loading plugin", zap.String("plugin_path", chp.PluginPath))
		chp.logger.Debug("handler-plugin loading plugin config", zap.Any("plugin_config", chp.PluginConfig))
		c, err := client.New(chp.PluginPath, chg.logger)
		if err != nil {
			return err
		}
		chp.client = c
		ok, err := chp.client.SetConfig(chp.PluginConfig)
		chp.logger.Debug("handler-plugin set plugin config", zap.Bool("ok", ok), zap.Error(err))
		if err != nil {
			return err
		}
		if !ok {
			chp.logger.Error("plugin `SetConfig` return false, maybe plugin not ready")
		}
		return nil
	}
}

func (chp *CaddyHandlerPlugin) unloadPlugin() error {
	if chp.client != nil {
		chp.client.Kill()
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var chp CaddyHandlerPlugin
	err := chp.UnmarshalCaddyfile(h.Dispenser)

	return &chp, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*CaddyHandlerPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*CaddyHandlerPlugin)(nil)
	_ caddyfile.Unmarshaler       = (*CaddyHandlerPlugin)(nil)
)
