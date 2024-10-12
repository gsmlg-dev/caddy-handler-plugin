package shared

import (
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-plugin"
)

type PluginQuery struct {
	Config    map[string][]string
	Method    string
	URL       url.URL
	URLBinary []byte
	Proto     string
	Host      string
	Header    http.Header
	Body      io.ReadCloser
}

func CreatePluginQuery(r *http.Request) PluginQuery {
	rUrl, _ := r.URL.MarshalBinary()
	return PluginQuery{
		Config:    r.URL.Query(),
		Method:    r.Method,
		URLBinary: rUrl,
		Proto:     r.Proto,
		Host:      r.Host,
		Header:    r.Header,
		Body:      r.Body,
	}
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
