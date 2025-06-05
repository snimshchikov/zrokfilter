package zrokfilter

import (
    "bytes"
    "io"
    "net/http"
    "os"

    "github.com/caddyserver/caddy/v2"
    "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
    "github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
    caddy.RegisterModule(ZrokFilter{})
}

type ZrokFilter struct {
    HTMLPath string `json:"html_path,omitempty"`
    body     []byte
}

func (ZrokFilter) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.zrokfilter",
        New: func() caddy.Module { return new(ZrokFilter) },
    }
}

func (zf *ZrokFilter) Provision(ctx caddy.Context) error {
    body, err := os.ReadFile(zf.HTMLPath)
    if err != nil {
        return err
    }
    zf.body = body
    return nil
}

func (zf ZrokFilter) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    rw := &interceptResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
    err := next.ServeHTTP(rw, r)
    if err != nil {
        return err
    }

    if rw.status == 404 && bytes.Contains(rw.buf.Bytes(), []byte("<title>zrok</title>")) {
        w.Header().Set("Content-Type", "text/html")
        w.WriteHeader(404)
        _, _ = w.Write(zf.body)
        return nil
    }

    w.WriteHeader(rw.status)
    _, _ = io.Copy(w, rw.buf)
    return nil
}

func (zf *ZrokFilter) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
    for d.Next() {
        for d.NextBlock(0) {
            switch d.Val() {
            case "html_path":
                if !d.NextArg() {
                    return d.ArgErr()
                }
                zf.HTMLPath = d.Val()
            }
        }
    }
    return nil
}

type interceptResponseWriter struct {
    http.ResponseWriter
    buf    *bytes.Buffer
    status int
}

func (rw *interceptResponseWriter) WriteHeader(code int) {
    rw.status = code
}

func (rw *interceptResponseWriter) Write(b []byte) (int, error) {
    return rw.buf.Write(b)
}
