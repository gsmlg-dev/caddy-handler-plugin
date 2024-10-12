package shared

import (
	"io"
	"net/http"
	"net/url"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
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
	Body             []byte
}

func CreatePluginQuery(r *http.Request) PluginQuery {
	body, _ := io.ReadAll(r.Body)

	return PluginQuery{
		Method:           r.Method,
		URL:              r.URL,
		Proto:            r.Proto,
		Host:             r.Host,
		Header:           r.Header,
		RemoteAddr:       r.RemoteAddr,
		TransferEncoding: r.TransferEncoding,
		RequestURI:       r.RequestURI,
		Body:             body,
	}
}

type PluginReply struct {
	Done   bool
	Status int
	Header http.Header
	Body   []byte
}

func (c *PluginReply) Serve(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if c.Done {
		if c.Status > 0 {
			w.WriteHeader(c.Status)
		}
		for k, v := range c.Header {
			for i, v := range v {
				if i == 0 {
					w.Header().Set(k, v)
				} else {
					w.Header().Add(k, v)
				}
			}
		}
		w.Write(c.Body)
		return nil
	}
	return next.ServeHTTP(w, r)
}

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "STATIC_PLUGIN",
	MagicCookieValue: "static-website-content",
}

var PluginMap = map[string]plugin.Plugin{
	"handler": &HandlerPlugin{},
}
