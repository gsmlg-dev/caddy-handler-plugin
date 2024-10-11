package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type Handler interface {
	Serve(r PluginQuery, w *PluginReply) error
}

type HandlerRPC struct{ client *rpc.Client }

func (g *HandlerRPC) Serve(q PluginQuery) (*PluginReply, error) {
	reply := &PluginReply{
		Done: false,
	}
	err := g.client.Call("Plugin.Serve", q, reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

type HandlerRPCServer struct {
	Impl Handler
}

func (s *HandlerRPCServer) Serve(r PluginQuery, reply *PluginReply) error {
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
