package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gsmlg-dev/caddy-handler-plugin/server"
	"github.com/gsmlg-dev/caddy-handler-plugin/shared"
)

var BuildTime string

const FileDir = "dist"

//go:embed all:dist/*
var buildFs embed.FS

var (
	defaultIndexNames = []string{"index.html", "index.htm", "index.txt"}
	defaultFileSuffix = []string{"html", "htm", "txt"}
)

type HandlerServer struct {
	server.HandlerServerDefault
	PassThru   bool
	IndexNames []string
	FileSuffix []string
}

func (g *HandlerServer) SetConfig(cfg map[string][]string, ok *bool) error {
	g.Config = cfg
	if _, has := cfg["pass_next"]; has {
		g.PassThru = true
	} else {
		g.PassThru = false
	}
	if names, has := cfg["index_names"]; has {
		g.IndexNames = names
	} else {
		g.IndexNames = defaultIndexNames
	}
	if suffix, has := cfg["file_suffix"]; has {
		g.FileSuffix = suffix
	} else {
		g.FileSuffix = defaultFileSuffix
	}
	*ok = true
	return nil
}

func (g *HandlerServer) Serve(q shared.PluginQuery, reply *shared.PluginReply) error {
	p := q.URL.Path
	filename := filepath.Join(FileDir, p)

	file, err := g.FindFile(filename)
	if err != nil {
		return g.NotFound(reply)
	}
	fileInfo, _ := file.Stat()

	header := http.Header{}
	header.Set("X-Served-By", "Static-Plugin")
	mime_typ := getMimeTypeByFileName(fileInfo.Name())
	header.Set("Content-Type", mime_typ)
	matched := setLastModified(&header, q.Header.Get("If-Modified-Since"))
	if matched {
		reply.Status = http.StatusNotModified
	} else {
		contents, _ := io.ReadAll(file)
		reply.Body = contents
	}
	reply.Header = header
	reply.Done = true
	return nil
}

func (g *HandlerServer) NotFound(reply *shared.PluginReply) error {
	if g.PassThru {
		reply.Done = false
		return nil
	}
	reply.Done = true
	reply.Status = 404
	reply.Body = []byte("Not found")
	return nil
}

func (g *HandlerServer) FindFile(p string) (fs.File, error) {
	p = strings.TrimRight(p, "/")
	f, err := buildFs.Open(p)
	var info fs.FileInfo
	if err == nil {
		info, _ = f.Stat()
		if !info.IsDir() {
			return f, nil
		}
	}
	for _, indexPage := range g.IndexNames {
		indexPath := filepath.Join(p, indexPage)

		f, err = buildFs.Open(indexPath)
		if err == nil {
			info, _ = f.Stat()
			if !info.IsDir() {
				return f, nil
			}
		}
	}
	for _, suffix := range g.FileSuffix {
		filePath := fmt.Sprintf("%s.%s", p, suffix)

		f, err = buildFs.Open(filePath)
		if err == nil {
			info, _ = f.Stat()
			if !info.IsDir() {
				return f, nil
			}
		}
	}
	return f, os.ErrNotExist
}

func main() {
	handler := &HandlerServer{
		PassThru: false,
	}
	server.New(handler)
}

func getMimeTypeByFileName(fileName string) string {
	// Extract the file extension (e.g., ".jpg")
	ext := filepath.Ext(fileName)
	// Get the MIME type based on the file extension
	mimeType := mime.TypeByExtension(ext)
	// If mimeType is empty, return a default or handle it as needed
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

func setLastModified(h *http.Header, if_mod_time string) bool {
	if BuildTime == "" {
		return false
	}
	i, err := strconv.ParseInt(BuildTime, 10, 64)
	if err == nil {
		tm := time.Unix(i, 0)
		mod_time := tm.UTC().Format(http.TimeFormat)
		h.Set("Last-Modified", mod_time)
		if mod_time == if_mod_time && if_mod_time != "" {
			return true
		}
	}
	return false
}
