package web

import (
	"net/http"
	"os"

	"mytun/internal/webdist"
)

// WebRoot 为空时使用内嵌产物；非空时（dev 模式 --web-root）从目录读文件。
// 对应目录结构：<root>/index.html 与 <root>/mytun-web.js。
var WebRoot = ""

// serveFile 优先从 WebRoot 读文件，回退到内嵌产物。
func serveFile(w http.ResponseWriter, r *http.Request, name, contentType string) {
	if WebRoot != "" {
		if b, err := os.ReadFile(WebRoot + "/" + name); err == nil {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
			return
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(webdist.MustAsset(name))
}
