// Package web 提供 Gin 路由与 handler：只做绑定校验+委托+信封，判定下沉。
package web

import (
	"mytun/internal/webdist"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// WebRoot 为空时使用内嵌产物；非空时（dev 模式 --web-root）从目录读文件。
// 对应目录结构：<root>/index.html 与 <root>/mytun-web.js。
var WebRoot = ""

// serveFile 优先从 WebRoot 读文件，回退到内嵌产物。
func serveFile(c *gin.Context, name, contentType string) {
	if WebRoot != "" {
		if b, err := os.ReadFile(WebRoot + "/" + name); err == nil {
			c.Data(http.StatusOK, contentType, b)
			return
		}
	}
	c.Data(http.StatusOK, contentType, webdist.MustAsset(name))
}
