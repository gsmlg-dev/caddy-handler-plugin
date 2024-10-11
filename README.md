# caddy-handler-plugin

Build go-plugin for caddy handler

## Usage

Create handler plugin

`hanlder.go`

```go
package main

import (
    "net/http"    
    
    "github.com/gsmlg-dev/caddy-handler-plugin/shared"
    "github.com/gsmlg-dev/caddy-handler-plugin/server"
)

type HandlerServer struct {
}

func (g *HandlerServer) Serve(r http.Request, reply *shared.PluginReply) error {
  reply.Done = true
  reply.Header.Set("Content-Type", "text/plain")
  reply.Body = []byte("Hello World")
  return nil
}

main() {
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
    route {
        handler_plugin {
            plugin_path "hanlder.bin"
        }
    }
}
```
