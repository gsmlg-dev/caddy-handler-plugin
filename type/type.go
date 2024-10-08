module staticplugintype

import (
	"embed"
)

type StaticFS struct {
	FS embed.FS
	Etag string
}

