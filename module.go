package staticplugin

import (
	"bytes"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"go.uber.org/zap"

	plugintype "github.com/gsmlg-dev/caddy-static-plugin/type"
	"github.com/hashicorp/go-plugin"
)

const DirectiveName = "static_plugin"

func init() {
	httpcaddyfile.RegisterHandlerDirective(DirectiveName, parseCaddyfile)

	caddy.RegisterModule(StaticPlugin{})
}

// StaticPlugin implements a static file server responder for Caddy.
type StaticPlugin struct {
	PluginPath string `json:"plugin_path,omitempty"`
	staticFS   *plugintype.StaticFS
	// A list of files or folders to hide; the file server will pretend as if
	// they don't exist. Accepts globular patterns like `*.ext` or `/foo/*/bar`
	// as well as placeholders. Because site roots can be dynamic, this list
	// uses file system paths, not request paths. To clarify, the base of
	// relative paths is the current working directory, NOT the site root.
	//
	// Entries without a path separator (`/` or `\` depending on OS) will match
	// any file or directory of that name regardless of its path. To hide only a
	// specific file with a name that may not be unique, always use a path
	// separator. For example, to hide all files or folder trees named "hidden",
	// put "hidden" in the list. To hide only ./hidden, put "./hidden" in the list.
	//
	// When possible, all paths are resolved to their absolute form before
	// comparisons are made. For maximum clarity and explictness, use complete,
	// absolute paths; or, for greater portability, use relative paths instead.
	Hide []string `json:"hide,omitempty"`

	// The names of files to try as index files if a folder is requested.
	IndexNames []string `json:"index_names,omitempty"`

	// Append suffix to request filename if origin name is not exists.
	SuffixNames []string `json:"suffix_names,omitempty"`

	// Enables file listings if a directory was requested and no index
	// file is present.
	// Browse *Browse `json:"browse,omitempty"`

	// Use redirects to enforce trailing slashes for directories, or to
	// remove trailing slash from URIs for files. Default is true.
	//
	// Canonicalization will not happen if the last element of the request's
	// path (the filename) is changed in an internal rewrite, to avoid
	// clobbering the explicit rewrite with implicit behavior.
	CanonicalURIs *bool `json:"canonical_uris,omitempty"`

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (StaticPlugin) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers." + DirectiveName,
		New: func() caddy.Module { return new(StaticPlugin) },
	}
}

// Provision sets up the static files responder.
func (fsrv *StaticPlugin) Provision(ctx caddy.Context) error {
	fsrv.logger = ctx.Logger(fsrv)

	if fsrv.IndexNames == nil {
		fsrv.IndexNames = defaultIndexNames
	}
	if fsrv.SuffixNames == nil {
		fsrv.SuffixNames = defaultSuffixNames
	}

	// for hide paths that are static (i.e. no placeholders), we can transform them into
	// absolute paths before the server starts for very slight performance improvement
	for i, h := range fsrv.Hide {
		if !strings.Contains(h, "{") && strings.Contains(h, separator) {
			if abs, err := filepath.Abs(h); err == nil {
				fsrv.Hide[i] = abs
			}
		}
	}

	return fsrv.openPlugin()
}

func (fsrv *StaticPlugin) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	filesToHide := fsrv.transformHidePaths(repl)

	// PathUnescape returns an error if the escapes aren't well-formed,
	// meaning the count % matches the RFC. Return early if the escape is
	// improper.
	if _, err := url.PathUnescape(r.URL.Path); err != nil {
		fsrv.logger.Debug("improper path escape",
			zap.String("request_path", r.URL.Path),
			zap.Error(err))
		return err
	}
	filename := "build" + r.URL.Path

	fsrv.logger.Debug("sanitized path join",
		zap.String("request_path", r.URL.Path),
		zap.String("result", filename))

	// get information about the file
	opF, err := fsrv.staticFS.FS.Open(filename)
	if err != nil {
		fsrv.logger.Debug("open file error",
			zap.String("error", err.Error()),
			zap.String("File", fmt.Sprintf("%v", opF)),
			zap.String("FileSystem", fmt.Sprintf("%v", fsrv.staticFS.FS)))
		err = mapDirOpenError(err, filename)
		if os.IsNotExist(err) {
			var info fs.FileInfo
			if len(fsrv.IndexNames) > 0 {
				for _, indexPage := range fsrv.IndexNames {
					indexPage := repl.ReplaceAll(indexPage, "")
					indexPath := caddyhttp.SanitizedPathJoin(filename, indexPage)
					if fileHidden(indexPath, filesToHide) {
						// pretend this file doesn't exist
						fsrv.logger.Debug("hiding index file",
							zap.String("filename", indexPath),
							zap.Strings("files_to_hide", filesToHide))
						continue
					}

					opF, err = fsrv.staticFS.FS.Open(indexPath)
					if err != nil {
						continue
					}
					info, _ = opF.Stat()
					filename = indexPath
					// implicitIndexFile = true
					fsrv.logger.Debug("located index file", zap.String("filename", filename))
					break
				}
			}
			if info == nil && strings.HasSuffix(filename, "/") == false {
				suffixList := []string{"html", "htm", "txt"}
				for _, suffix := range suffixList {
					suffix := repl.ReplaceAll(suffix, "")
					filePath := fmt.Sprintf("%s.%s", filename, suffix)
					if fileHidden(filePath, filesToHide) {
						// pretend this file doesn't exist
						fsrv.logger.Debug("hiding index file",
							zap.String("filename", filePath),
							zap.Strings("files_to_hide", filesToHide))
						continue
					}

					opF, err = fsrv.staticFS.FS.Open(filePath)
					if err != nil {
						continue
					}
					info, _ = opF.Stat()
					filename = filePath
					// implicitIndexFile = true
					fsrv.logger.Debug("located file with suffix", zap.String("filename", filename), zap.String("suffix", suffix))
					break
				}
			}
			if info == nil {
				return fsrv.notFound(w, r, next)
			}
		} else if os.IsPermission(err) {
			return caddyhttp.Error(http.StatusForbidden, err)
		}
	}
	info, err := opF.Stat()
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	// if the request mapped to a directory, see if
	// there is an index file we can serve
	var implicitIndexFile bool
	if info.IsDir() && len(fsrv.IndexNames) > 0 {
		for _, indexPage := range fsrv.IndexNames {
			indexPage := repl.ReplaceAll(indexPage, "")
			indexPath := caddyhttp.SanitizedPathJoin(filename, indexPage)
			if fileHidden(indexPath, filesToHide) {
				// pretend this file doesn't exist
				fsrv.logger.Debug("hiding index file",
					zap.String("filename", indexPath),
					zap.Strings("files_to_hide", filesToHide))
				continue
			}

			opF, err := fsrv.staticFS.FS.Open(indexPath)
			if err != nil {
				continue
			}
			indexInfo, _ := opF.Stat()

			// don't rewrite the request path to append
			// the index file, because we might need to
			// do a canonical-URL redirect below based
			// on the URL as-is

			// we've chosen to use this index file,
			// so replace the last file info and path
			// with that of the index file
			info = indexInfo
			filename = indexPath
			implicitIndexFile = true
			fsrv.logger.Debug("located index file", zap.String("filename", filename))
			break
		}
	}

	// if still dir try to find out if it is a file with suffix
	if info.IsDir() && !strings.HasSuffix(filename, "/") {
		suffixList := fsrv.SuffixNames
		for _, suffix := range suffixList {
			suffix := repl.ReplaceAll(suffix, "")
			filePath := fmt.Sprintf("%s.%s", filename, suffix)
			if fileHidden(filePath, filesToHide) {
				// pretend this file doesn't exist
				fsrv.logger.Debug("hiding index file",
					zap.String("filename", filePath),
					zap.Strings("files_to_hide", filesToHide))
				continue
			}

			opF, err = fsrv.staticFS.FS.Open(filePath)
			if err != nil {
				continue
			}
			info, _ = opF.Stat()
			filename = filePath
			// implicitIndexFile = true
			fsrv.logger.Debug("located file with suffix", zap.String("filename", filename), zap.String("suffix", suffix))
			break
		}
	}

	// if still referencing a directory, delegate
	// to browse or return an error
	if info.IsDir() {
		fsrv.logger.Debug("no index file in directory",
			zap.String("path", filename),
			zap.Strings("index_filenames", fsrv.IndexNames))
		return fsrv.notFound(w, r, next)
	}

	// one last check to ensure the file isn't hidden (we might
	// have changed the filename from when we last checked)
	if fileHidden(filename, filesToHide) {
		fsrv.logger.Debug("hiding file",
			zap.String("filename", filename),
			zap.Strings("files_to_hide", filesToHide))
		return fsrv.notFound(w, r, next)
	}

	// if URL canonicalization is enabled, we need to enforce trailing
	// slash convention: if a directory, trailing slash; if a file, no
	// trailing slash - not enforcing this can break relative hrefs
	// in HTML (see https://github.com/caddyserver/caddy/issues/2741)
	if info == nil && (fsrv.CanonicalURIs == nil || *fsrv.CanonicalURIs) {
		// Only redirect if the last element of the path (the filename) was not
		// rewritten; if the admin wanted to rewrite to the canonical path, they
		// would have, and we have to be very careful not to introduce unwanted
		// redirects and especially redirect loops!
		// See https://github.com/caddyserver/caddy/issues/4205.
		origReq := r.Context().Value(caddyhttp.OriginalRequestCtxKey).(http.Request)
		if path.Base(origReq.URL.Path) == path.Base(r.URL.Path) {
			if implicitIndexFile && !strings.HasSuffix(origReq.URL.Path, "/") {
				to := origReq.URL.Path + "/"
				fsrv.logger.Debug("redirecting to canonical URI (adding trailing slash for directory)",
					zap.String("from_path", origReq.URL.Path),
					zap.String("to_path", to))
				return redirect(w, r, to)
			} else if !implicitIndexFile && strings.HasSuffix(origReq.URL.Path, "/") {
				to := origReq.URL.Path[:len(origReq.URL.Path)-1]
				fsrv.logger.Debug("redirecting to canonical URI (removing trailing slash for file)",
					zap.String("from_path", origReq.URL.Path),
					zap.String("to_path", to))
				return redirect(w, r, to)
			}
		}
	}

	var file []byte

	// no precompressed file found, use the actual file
	if file == nil {
		fsrv.logger.Debug("opening file", zap.String("filename", filename))

		// open the file
		file, err = fsrv.openFile(filename, w)
		if err != nil {
			if herr, ok := err.(caddyhttp.HandlerError); ok &&
				herr.StatusCode == http.StatusNotFound {
				return fsrv.notFound(w, r, next)
			}
			return err // error is already structured
		}
	}

	// set the ETag - note that a conditional If-None-Match request is handled
	// by http.ServeContent below, which checks against this ETag value
	if fsrv.staticFS.Etag != "" {
		w.Header().Set("ETag", fsrv.staticFS.Etag)
	} else {
		w.Header().Set("ETag", calculateEtag(info))
	}

	if w.Header().Get("Content-Type") == "" {
		mtyp := mime.TypeByExtension(filepath.Ext(filename))
		if mtyp == "" {
			// do not allow Go to sniff the content-type; see
			// https://www.youtube.com/watch?v=8t8JYpt0egE
			// TODO: If we want a Content-Type, consider writing a default of application/octet-stream - this is secure but violates spec
			w.Header()["Content-Type"] = nil
		} else {
			w.Header().Set("Content-Type", mtyp)
		}
	}

	// let the standard library do what it does best; note, however,
	// that errors generated by ServeContent are written immediately
	// to the response, so we cannot handle them (but errors there
	// are rare)
	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(file))

	return nil
}

// openPlugin opens the file at the given filename. If there was an error,
// the response is configured to inform the client how to best handle it
// and a well-described handler error is returned (do not wrap the
// returned error value).
func (fsrv *StaticPlugin) openPlugin() error {
	if fsrv.PluginPath == "" {
		return fmt.Errorf("plugin_path is required")
	} else {
		// handshakeConfigs are used to just do a basic handshake between
		// a plugin and host. If the handshake fails, a user friendly error is shown.
		// This prevents users from executing bad plugins or executing a plugin
		// directory. It is a UX feature, not a security feature.
		var handshakeConfig = plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "BASIC_PLUGIN",
			MagicCookieValue: "hello",
		}

		// pluginMap is the map of plugins we can dispense.
		var pluginMap = map[string]plugin.Plugin{}

		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: handshakeConfig,
			Plugins:         pluginMap,
			Cmd:             exec.Command(fsrv.PluginPath),
			// Logger:          fsrv.logger,
		})
		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			return err
		}

		// Request the plugin
		raw, err := rpcClient.Dispense("New")
		if err != nil {
			return err
		}

		// We should have a Greeter now! This feels like a normal interface
		// implementation but is in fact over an RPC connection.
		f := raw.(plugintype.StaticFS)
		fsrv.staticFS = &f
		return nil
	}
}

// openFile opens the file at the given filename. If there was an error,
// the response is configured to inform the client how to best handle it
// and a well-described handler error is returned (do not wrap the
// returned error value).
func (fsrv *StaticPlugin) openFile(filename string, w http.ResponseWriter) ([]byte, error) {
	file, err := fsrv.staticFS.FS.ReadFile(filename)
	if err != nil {
		err = mapDirOpenError(err, filename)
		if os.IsNotExist(err) {
			fsrv.logger.Debug("file not found", zap.String("filename", filename), zap.Error(err))
			return nil, caddyhttp.Error(http.StatusNotFound, err)
		} else if os.IsPermission(err) {
			fsrv.logger.Debug("permission denied", zap.String("filename", filename), zap.Error(err))
			return nil, caddyhttp.Error(http.StatusForbidden, err)
		}
		return nil, caddyhttp.Error(http.StatusServiceUnavailable, err)
	}
	return file, nil
}

// mapDirOpenError maps the provided non-nil error from opening name
// to a possibly better non-nil error. In particular, it turns OS-specific errors
// about opening files in non-directories into os.ErrNotExist. See golang/go#18984.
// Adapted from the Go standard library; originally written by Nathaniel Caza.
// https://go-review.googlesource.com/c/go/+/36635/
// https://go-review.googlesource.com/c/go/+/36804/
func mapDirOpenError(originalErr error, name string) error {
	if os.IsNotExist(originalErr) || os.IsPermission(originalErr) {
		return originalErr
	}

	parts := strings.Split(name, separator)
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := os.Stat(strings.Join(parts[:i+1], separator))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return os.ErrNotExist
		}
	}

	return originalErr
}

// transformHidePaths performs replacements for all the elements of fsrv.Hide and
// makes them absolute paths (if they contain a path separator), then returns a
// new list of the transformed values.
func (fsrv *StaticPlugin) transformHidePaths(repl *caddy.Replacer) []string {
	hide := make([]string, len(fsrv.Hide))
	for i := range fsrv.Hide {
		hide[i] = repl.ReplaceAll(fsrv.Hide[i], "")
		if strings.Contains(hide[i], separator) {
			abs, err := filepath.Abs(hide[i])
			if err == nil {
				hide[i] = abs
			}
		}
	}
	return hide
}

// fileHidden returns true if filename is hidden according to the hide list.
// filename must be a relative or absolute file system path, not a request
// URI path. It is expected that all the paths in the hide list are absolute
// paths or are singular filenames (without a path separator).
func fileHidden(filename string, hide []string) bool {
	if len(hide) == 0 {
		return false
	}

	// all path comparisons use the complete absolute path if possible
	filenameAbs, err := filepath.Abs(filename)
	if err == nil {
		filename = filenameAbs
	}

	var components []string

	for _, h := range hide {
		if !strings.Contains(h, separator) {
			// if there is no separator in h, then we assume the user
			// wants to hide any files or folders that match that
			// name; thus we have to compare against each component
			// of the filename, e.g. hiding "bar" would hide "/bar"
			// as well as "/foo/bar/baz" but not "/barstool".
			if len(components) == 0 {
				components = strings.Split(filename, separator)
			}
			for _, c := range components {
				if hidden, _ := filepath.Match(h, c); hidden {
					return true
				}
			}
		} else if strings.HasPrefix(filename, h) {
			// if there is a separator in h, and filename is exactly
			// prefixed with h, then we can do a prefix match so that
			// "/foo" matches "/foo/bar" but not "/foobar".
			withoutPrefix := strings.TrimPrefix(filename, h)
			if strings.HasPrefix(withoutPrefix, separator) {
				return true
			}
		}

		// in the general case, a glob match will suffice
		if hidden, _ := filepath.Match(h, filename); hidden {
			return true
		}
	}

	return false
}

// notFound returns a 404 error or, if pass-thru is enabled,
// it calls the next handler in the chain.
func (fsrv *StaticPlugin) notFound(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return next.ServeHTTP(w, r)
}

// parseCaddyfile parses the static_site directive. It enables the static file
// server and configures it with this syntax:
//
//	static_site [<matcher>] [browse] {
//	    hide          <files...>
//	    index         <files...>
//	    precompressed <formats...>
//	    status        <status>
//	    disable_canonical_uris
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var fsrv StaticPlugin

	for h.Next() {
		for h.NextBlock(0) {
			switch h.Val() {
			case "plugin_path":
				pluginPath := h.RemainingArgs()
				if len(fsrv.PluginPath) != 1 {
					return nil, h.ArgErr()
				} else {
					fsrv.PluginPath = pluginPath[0]
				}

			case "hide":
				fsrv.Hide = h.RemainingArgs()
				if len(fsrv.Hide) == 0 {
					return nil, h.ArgErr()
				}

			case "index":
				fsrv.IndexNames = h.RemainingArgs()
				if len(fsrv.IndexNames) == 0 {
					return nil, h.ArgErr()
				}

			case "suffix":
				fsrv.SuffixNames = h.RemainingArgs()
				if len(fsrv.SuffixNames) == 0 {
					return nil, h.ArgErr()
				}

			case "disable_canonical_uris":
				if h.NextArg() {
					return nil, h.ArgErr()
				}
				falseBool := false
				fsrv.CanonicalURIs = &falseBool

			default:
				return nil, h.Errf("unknown subdirective '%s'", h.Val())
			}
		}
	}

	// hide the Caddyfile (and any imported Caddyfiles)
	if configFiles := h.Caddyfiles(); len(configFiles) > 0 {
		for _, file := range configFiles {
			file = filepath.Clean(file)
			if !fileHidden(file, fsrv.Hide) {
				// if there's no path separator, the file server module will hide all
				// files by that name, rather than a specific one; but we want to hide
				// only this specific file, so ensure there's always a path separator
				if !strings.Contains(file, separator) {
					file = "." + separator + file
				}
				fsrv.Hide = append(fsrv.Hide, file)
			}
		}
	}

	return &fsrv, nil
}

// calculateEtag produces a strong etag by default, although, for
// efficiency reasons, it does not actually consume the contents
// of the file to make a hash of all the bytes. ¯\_(ツ)_/¯
// Prefix the etag with "W/" to convert it into a weak etag.
// See: https://tools.ietf.org/html/rfc7232#section-2.3
func calculateEtag(d os.FileInfo) string {
	t := strconv.FormatInt(d.ModTime().Unix(), 36)
	s := strconv.FormatInt(d.Size(), 36)
	return `"` + t + s + `"`
}

func redirect(w http.ResponseWriter, r *http.Request, to string) error {
	for strings.HasPrefix(to, "//") {
		// prevent path-based open redirects
		to = strings.TrimPrefix(to, "/")
	}
	http.Redirect(w, r, to, http.StatusPermanentRedirect)
	return nil
}

var defaultIndexNames = []string{"index.html", "index.htm", "index.txt"}

var defaultSuffixNames = []string{"html", "htm", "txt"}

const (
	minBackoff, maxBackoff = 2, 5
	separator              = string(filepath.Separator)
)

// Interface guards
var (
	_ caddy.Provisioner           = (*StaticPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*StaticPlugin)(nil)
)
