// Package web 提供标准库 HTTP 路由与 handler：只做绑定校验+委托+信封，判定下沉。
package web

import (
	"errors"
	"net/http"
	"net/netip"
	"strconv"

	"mytun/internal/packet"
	"mytun/internal/tunmgr"
)

// Manager 是 handler 需要的 tunmgr 能力子集，便于单测 fake。
type Manager interface {
	Create(addr string, mtu int) (string, error)
	Delete() error
	Status() (tunmgr.State, string, string, int, tunmgr.Stats)
	PushLog(dir, summary string)
	Logs(limit int) []tunmgr.LogEntry
	AddTx(n int)
	Device() tunmgr.Device
}

// Server 持有 Manager 与 version，供各 handler 使用。
type Server struct {
	m       Manager
	version string
}

// NewRouter 构造标准库 ServeMux。version 写入 status/version 字段。
func NewRouter(m Manager, version string) *http.ServeMux {
	s := &Server{m: m, version: version}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "index.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("GET /mytun-web.js", func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "mytun-web.js", "application/javascript")
	})
	mux.HandleFunc("GET /style.css", func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "style.css", "text/css; charset=utf-8")
	})
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/tun", s.handleCreateTun)
	mux.HandleFunc("DELETE /api/tun", s.handleDeleteTun)
	mux.HandleFunc("POST /api/send", s.handleSend)
	mux.HandleFunc("GET /api/logs", s.handleLogs)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	state, ifname, addr, mtu, st := s.m.Status()
	ok(w, map[string]any{
		"state":  string(state),
		"ifname": ifname,
		"addr":   addr,
		"mtu":    mtu,
		"stats": map[string]any{
			"rxPackets": st.RxPackets,
			"txPackets": st.TxPackets,
			"rxBytes":   st.RxBytes,
			"txBytes":   st.TxBytes,
		},
		"version": s.version,
	})
}

func (s *Server) handleCreateTun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Addr string `json:"addr"`
		MTU  int    `json:"mtu"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "bad json: "+err.Error())
		return
	}
	pfx, err := netip.ParsePrefix(req.Addr)
	if err != nil || !pfx.Addr().Is4() {
		// 普通 IPv4 地址（无掩码）也接受，默认 /24。
		a, err2 := netip.ParseAddr(req.Addr)
		if err2 != nil || !a.Is4() {
			fail(w, http.StatusBadRequest, "BAD_ADDR", "invalid addr: "+req.Addr)
			return
		}
	}
	ifname, err := s.m.Create(req.Addr, req.MTU)
	if err != nil {
		if errors.Is(err, tunmgr.ErrExists) {
			fail(w, http.StatusConflict, "TUN_EXISTS", "tun already exists")
			return
		}
		fail(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}
	ok(w, map[string]any{"ifname": ifname})
}

func (s *Server) handleDeleteTun(w http.ResponseWriter, r *http.Request) {
	if err := s.m.Delete(); err != nil {
		if errors.Is(err, tunmgr.ErrNoTun) {
			fail(w, http.StatusNotFound, "NO_TUN", "no tun")
			return
		}
		fail(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	ok(w, map[string]any{"deleted": true})
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dst  string `json:"dst"`
		Size int    `json:"size"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "bad json: "+err.Error())
		return
	}
	dst, err := netip.ParseAddr(req.Dst)
	if err != nil || !dst.Is4() {
		fail(w, http.StatusBadRequest, "BAD_DST", "invalid dst: "+req.Dst)
		return
	}
	size := req.Size
	if size == 0 {
		size = 64
	}
	state, _, addrStr, _, _ := s.m.Status()
	if state != tunmgr.StateUp {
		fail(w, http.StatusNotFound, "NO_TUN", "no tun")
		return
	}
	srcAddr, err := prefixAddr(addrStr)
	if err != nil {
		fail(w, http.StatusInternalServerError, "BAD_SRC", err.Error())
		return
	}
	pkt, err := packet.BuildIPv4EchoRequest(srcAddr, dst, size)
	if err != nil {
		fail(w, http.StatusBadRequest, "BAD_SIZE", err.Error())
		return
	}
	dev := s.m.Device()
	if dev == nil {
		fail(w, http.StatusNotFound, "NO_TUN", "no tun")
		return
	}
	n, err := dev.Write(pkt)
	if err != nil {
		fail(w, http.StatusInternalServerError, "WRITE_FAILED", err.Error())
		return
	}
	s.m.AddTx(n)
	s.m.PushLog("tx", packet.ParseSummary(pkt))
	ok(w, map[string]any{"len": n})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if str := r.URL.Query().Get("limit"); str != "" {
		n, err := strconv.Atoi(str)
		if err != nil || n < 0 {
			fail(w, http.StatusBadRequest, "BAD_LIMIT", "invalid limit: "+str)
			return
		}
		limit = n
	}
	ok(w, map[string]any{"logs": s.m.Logs(limit)})
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
