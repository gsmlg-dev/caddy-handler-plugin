# caddy-static-plugin

Build static website into go plugin, load plugin in caddy

## Usage

Create website plugin

`website.go`

```go
package main

import (
    "time"
    "fmt"
    "embed"
    "github.com/gsmlg-dev/caddy-static-plugin/type"
)

//go:embed website/*
var websiteFS embed.FS

func New() *type.StaticFS {
    time := time.Now().Unit()

    return &type.StaticFS{
        FS: websiteFS,
        Etag: fmt.Sprintf("%s", time),
    }
}
```

```shell
# Build Plugin
go build -buildmode=plugin -o website-plugin.so website.go
```

Built `Caddy` with `caddy-static-plugin`

```shell
xcaddy build --with github.com/gsmlg-dev/caddy-static-plugin=/tmp/caddy-static-plugin
```

Load plugin in `Caddyfile`

```caddyfile
localhost:8080 {
    static_plugin {
        plugin_path "website-plugin.so"
    }
}
```

## *Warning*

go plugin has it's limition, it can't be used in some situation, such as:

- `CGO_ENABLED=0` is not supported
- Needs build with same go version
- Other go plugin limition
