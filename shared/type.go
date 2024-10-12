package shared

import (
	"net/http"
	"net/url"

	"github.com/hashicorp/go-plugin"
)

type PluginQuery struct {
	Method           string
	URL              *url.URL
	Proto            string
	Host             string
	Header           http.Header
	RemoteAddr       string
	TransferEncoding []string
	RequestURI       string
	// Body             []byte
}

func CreatePluginQuery(r *http.Request) PluginQuery {
	// var body []byte
	// r.Body.Read(body)

	return PluginQuery{
		Method:           r.Method,
		URL:              r.URL,
		Proto:            r.Proto,
		Host:             r.Host,
		Header:           r.Header,
		RemoteAddr:       r.RemoteAddr,
		TransferEncoding: r.TransferEncoding,
		RequestURI:       r.RequestURI,
		// Body:             body,
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
