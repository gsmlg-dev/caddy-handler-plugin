package shared

import (
	"net/http"

	"github.com/hashicorp/go-plugin"
)

type PluginQuery struct {
	Config map[string][]string
	Path   string
	Header http.Header
}

type PluginReply struct {
	Done   bool
	Status int
	Header http.Header
	Body   []byte
}

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "STATIC_PLUGIN",
	MagicCookieValue: "static-website-content",
}

var PluginMap = map[string]plugin.Plugin{
	"handler": &HandlerPlugin{},
}
