// Package web 提供 Gin 路由与 handler：只做绑定校验+委托+信封，判定下沉。
package web

import (
	"errors"
	"net/http"
	"net/netip"
	"strconv"

	"github.com/gin-gonic/gin"

	"mytun/internal/packet"
	"mytun/internal/tunmgr"
)

// indexPlaceholder 是 `/` 的占位页面，Task 5 替换为 embed 打包产物。
const indexPlaceholder = `<!doctype html><html><head><meta charset="utf-8"><title>mytun</title></head><body><h1>mytun web panel</h1></body></html>`

// NewRouter 构造 Gin 引擎。version 写入 status/version 字段。
func NewRouter(m *tunmgr.Manager, version string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		serveFile(c, "index.html", "text/html; charset=utf-8")
	})
	r.GET("/mytun-web.js", func(c *gin.Context) {
		serveFile(c, "mytun-web.js", "application/javascript")
	})
	r.GET("/style.css", func(c *gin.Context) {
		serveFile(c, "style.css", "text/css; charset=utf-8")
	})
	r.GET("/healthz", func(c *gin.Context) {
		ok(c, gin.H{"status": "ok"})
	})
	r.GET("/api/status", func(c *gin.Context) {
		state, ifname, addr, mtu, s := m.Status()
		ok(c, gin.H{
			"state":  string(state),
			"ifname": ifname,
			"addr":   addr,
			"mtu":    mtu,
			"stats": gin.H{
				"rxPackets": s.RxPackets,
				"txPackets": s.TxPackets,
				"rxBytes":   s.RxBytes,
				"txBytes":   s.TxBytes,
			},
			"version": version,
		})
	})

	r.POST("/api/tun", func(c *gin.Context) {
		var req struct {
			Addr string `json:"addr"`
			MTU  int    `json:"mtu"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "BAD_REQUEST", "bad json: "+err.Error())
			return
		}
		pfx, err := netip.ParsePrefix(req.Addr)
		if err != nil || !pfx.Addr().Is4() {
			// 普通 IPv4 地址（无掩码）也接受，默认 /24。
			a, err2 := netip.ParseAddr(req.Addr)
			if err2 != nil || !a.Is4() {
				fail(c, http.StatusBadRequest, "BAD_ADDR", "invalid addr: "+req.Addr)
				return
			}
		}
		ifname, err := m.Create(req.Addr, req.MTU)
		if err != nil {
			if errors.Is(err, tunmgr.ErrExists) {
				fail(c, http.StatusConflict, "TUN_EXISTS", "tun already exists")
				return
			}
			fail(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
			return
		}
		ok(c, gin.H{"ifname": ifname})
	})

	r.DELETE("/api/tun", func(c *gin.Context) {
		if err := m.Delete(); err != nil {
			if errors.Is(err, tunmgr.ErrNoTun) {
				fail(c, http.StatusNotFound, "NO_TUN", "no tun")
				return
			}
			fail(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
			return
		}
		ok(c, gin.H{"deleted": true})
	})

	r.POST("/api/send", func(c *gin.Context) {
		var req struct {
			Dst  string `json:"dst"`
			Size int    `json:"size"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "BAD_REQUEST", "bad json: "+err.Error())
			return
		}
		dst, err := netip.ParseAddr(req.Dst)
		if err != nil || !dst.Is4() {
			fail(c, http.StatusBadRequest, "BAD_DST", "invalid dst: "+req.Dst)
			return
		}
		size := req.Size
		if size == 0 {
			size = 64
		}
		state, _, addrStr, _, _ := m.Status()
		if state != tunmgr.StateUp {
			fail(c, http.StatusNotFound, "NO_TUN", "no tun")
			return
		}
		srcAddr, err := prefixAddr(addrStr)
		if err != nil {
			fail(c, http.StatusInternalServerError, "BAD_SRC", err.Error())
			return
		}
		pkt, err := packet.BuildIPv4EchoRequest(srcAddr, dst, size)
		if err != nil {
			fail(c, http.StatusBadRequest, "BAD_SIZE", err.Error())
			return
		}
		dev := m.Device()
		if dev == nil {
			fail(c, http.StatusNotFound, "NO_TUN", "no tun")
			return
		}
		n, err := dev.Write(pkt)
		if err != nil {
			fail(c, http.StatusInternalServerError, "WRITE_FAILED", err.Error())
			return
		}
		m.AddTx(n)
		m.PushLog("tx", packet.ParseSummary(pkt))
		ok(c, gin.H{"len": n})
	})

	r.GET("/api/logs", func(c *gin.Context) {
		limit := 0
		if s := c.Query("limit"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 0 {
				fail(c, http.StatusBadRequest, "BAD_LIMIT", "invalid limit: "+s)
				return
			}
			limit = n
		}
		ok(c, gin.H{"logs": m.Logs(limit)})
	})

	return r
}

// prefixAddr 从 "10.0.0.1/24" 或 "10.0.0.1" 中取出 IPv4 地址。
func prefixAddr(s string) (netip.Addr, error) {
	if pfx, err := netip.ParsePrefix(s); err == nil {
		return pfx.Addr(), nil
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, errors.New("invalid configured addr: " + s)
	}
	return a, nil
}
