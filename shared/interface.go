package shared

import (
	"net/http"
	"net/rpc"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/hashicorp/go-plugin"
)

type Handler interface {
	Serve(r http.Request, w *PluginReply) error
}

type HandlerRPC struct{ client *rpc.Client }

func (g *HandlerRPC) Serve(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	reply := &PluginReply{
		Done: false,
	}
	err := g.client.Call("Plugin.Serve", r, reply)
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

type HandlerRPCServer struct {
	Impl Handler
}

func (s *HandlerRPCServer) Serve(r http.Request, reply *PluginReply) error {
	return s.Impl.Serve(r, reply)
}

type HandlerPlugin struct {
	Impl Handler
}

func (p *HandlerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &HandlerRPCServer{Impl: p.Impl}, nil
}

func (HandlerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &HandlerRPC{client: c}, nil
}
