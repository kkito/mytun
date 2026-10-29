package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ok 返回统一成功信封 {ok:true,data}。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

// fail 返回统一失败信封 {ok:false,error:{code,message}}。
func fail(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{
		"ok":    false,
		"error": gin.H{"code": code, "message": msg},
	})
}
