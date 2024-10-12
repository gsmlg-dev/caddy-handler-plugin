# caddy-handler-plugin

Build go-plugin for caddy handler

## Usage

Create handler plugin

`hanlder.go`

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/gsmlg-dev/caddy-handler-plugin/shared"
    "github.com/gsmlg-dev/caddy-handler-plugin/server"
)

type HandlerServer struct {
    server.HandlerServerDefault
}

func (g *HandlerServer) Serve(q shared.PluginQuery, reply *shared.PluginReply) error {
  reply.Done = true
  header := http.Header{}
  header.Set("Server-Handler", "Custom Caddy Handler")
  header.Set("Content-Type", "text/plain")
  reply.Header = header
  out := fmt.Sprintf(`Hello World

  * with Query:

  %v

  * with Config:

  %v
`, q, g.Config)
  reply.Body = []byte(out)
  return nil
}

func main() {
        handler := &HandlerServer{}
        server.New(handler)
}
```

```shell
# Build Plugin
go build -o hanlder.bin hanlder.go
```

Built `Caddy` with `caddy-handler-plugin`

```shell
xcaddy build --with github.com/gsmlg-dev/caddy-handler-plugin
```

Load plugin in `Caddyfile`

```caddyfile
localhost:8080 {
    handler_plugin "/caddy-plugins/hanlder.bin" {
        name "web handler"
        pass_next_if_not_match false
    }
}
```
