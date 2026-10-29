package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"mytun/internal/tunmgr"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeDevice struct {
	mu       sync.Mutex
	name     string
	closed   bool
	written  [][]byte
	writeErr error
}

func (f *fakeDevice) Read(p []byte) (int, error) { return 0, errors.New("no packet") }
func (f *fakeDevice) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	cp := append([]byte(nil), p...)
	f.written = append(f.written, cp)
	return len(p), nil
}
func (f *fakeDevice) ActualName() (string, error) { return f.name, nil }
func (f *fakeDevice) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}
func (f *fakeDevice) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.written)
}

func newTestRouter() (*tunmgr.Manager, *fakeDevice, *gin.Engine) {
	dev := &fakeDevice{name: "utun9"}
	m := tunmgr.New(dev, func(ifname string) error { return nil })
	return m, dev, NewRouter(m, "test")
}

func doReq(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("response is not JSON: %v: %q", err, w.Body.String())
	}
	return v
}

func TestHealthz(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "GET", "/healthz", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", w.Code)
	}
	v := decodeBody(t, w)
	if v["ok"] != true {
		t.Fatalf("healthz ok = %v, want true: %v", v["ok"], v)
	}
}

func TestStatusDownInitially(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "GET", "/api/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/status = %d, want 200", w.Code)
	}
	v := decodeBody(t, w)
	data := v["data"].(map[string]any)
	if data["state"] != "down" {
		t.Fatalf("state = %v, want down", data["state"])
	}
}

func TestCreateTunBadAddr(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "POST", "/api/tun", `{"addr":"not-an-ip","mtu":1420}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/tun bad addr = %d, want 400", w.Code)
	}
	v := decodeBody(t, w)
	if v["ok"] != false {
		t.Fatalf("ok = %v, want false", v["ok"])
	}
}

func TestCreateTunOK(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "POST", "/api/tun", `{"addr":"10.0.0.1/24","mtu":1420}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/tun = %d, want 200: %s", w.Code, w.Body.String())
	}
	w2 := doReq(r, "GET", "/api/status", "")
	data := decodeBody(t, w2)["data"].(map[string]any)
	if data["state"] != "up" {
		t.Fatalf("state = %v, want up", data["state"])
	}
	if data["addr"] != "10.0.0.1/24" {
		t.Fatalf("addr = %v", data["addr"])
	}
}

func TestCreateTunDuplicate409(t *testing.T) {
	_, _, r := newTestRouter()
	doReq(r, "POST", "/api/tun", `{"addr":"10.0.0.1/24","mtu":1420}`)
	w := doReq(r, "POST", "/api/tun", `{"addr":"10.0.0.2/24","mtu":1420}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate POST /api/tun = %d, want 409", w.Code)
	}
	v := decodeBody(t, w)
	errObj := v["error"].(map[string]any)
	if errObj["code"] != "TUN_EXISTS" {
		t.Fatalf("code = %v, want TUN_EXISTS", errObj["code"])
	}
}

func TestDeleteTunOK(t *testing.T) {
	_, _, r := newTestRouter()
	doReq(r, "POST", "/api/tun", `{"addr":"10.0.0.1/24","mtu":1420}`)
	w := doReq(r, "DELETE", "/api/tun", "")
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /api/tun = %d, want 200: %s", w.Code, w.Body.String())
	}
	w2 := doReq(r, "GET", "/api/status", "")
	data := decodeBody(t, w2)["data"].(map[string]any)
	if data["state"] != "down" {
		t.Fatalf("state = %v, want down", data["state"])
	}
}

func TestDeleteTunNoTun404(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "DELETE", "/api/tun", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("DELETE /api/tun without tun = %d, want 404", w.Code)
	}
	v := decodeBody(t, w)
	errObj := v["error"].(map[string]any)
	if errObj["code"] != "NO_TUN" {
		t.Fatalf("code = %v, want NO_TUN", errObj["code"])
	}
}

func TestSendNoTun404(t *testing.T) {
	_, _, r := newTestRouter()
	w := doReq(r, "POST", "/api/send", `{"dst":"10.0.0.2"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("POST /api/send without tun = %d, want 404", w.Code)
	}
}

func TestSendOK(t *testing.T) {
	_, dev, r := newTestRouter()
	doReq(r, "POST", "/api/tun", `{"addr":"10.0.0.1/24","mtu":1420}`)
	w := doReq(r, "POST", "/api/send", `{"dst":"10.0.0.2","size":64}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/send = %d, want 200: %s", w.Code, w.Body.String())
	}
	if dev.writeCount() != 1 {
		t.Fatalf("device writes = %d, want 1", dev.writeCount())
	}
	v := decodeBody(t, w)
	data := v["data"].(map[string]any)
	if _, ok := data["len"]; !ok {
		t.Fatalf("send data missing len: %v", data)
	}
}

func TestLogsLimit(t *testing.T) {
	m, _, r := newTestRouter()
	for i := 0; i < 5; i++ {
		m.PushLog("tx", "pkt")
	}
	w := doReq(r, "GET", "/api/logs?limit=2", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/logs = %d, want 200", w.Code)
	}
	data := decodeBody(t, w)["data"].(map[string]any)
	logs := data["logs"].([]any)
	if len(logs) != 2 {
		t.Fatalf("len(logs) = %d, want 2", len(logs))
	}
}
