## Example Static Plugin

Build Caddy with

```shell
xcaddy build --with github.com/gsmlg-dev/caddy-handler-plugin
```

Build This Plugin

```shell
go build -ldflags "-X main.BuildTime=$(date +%s)" -o static.bin main.go
```

Run `caddy` with `Caddyfile`

```caddyfile
:8080 {
    route {
        handler_plugin * "/sites/static.bin" {
            index_names "index.html" "index.txt"
            file_suffix "html" "txt"
        }
    }
}
```
