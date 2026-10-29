package tunmgr

import (
	"errors"
	"sync"
	"testing"
)

// fakeDevice 实现 Device 接口注入，不建真实 TUN。
type fakeDevice struct {
	mu     sync.Mutex
	name   string
	closed bool
}

func (f *fakeDevice) Read(p []byte) (int, error)  { return 0, errors.New("no packet") }
func (f *fakeDevice) Write(p []byte) (int, error) { return len(p), nil }
func (f *fakeDevice) ActualName() (string, error) { return f.name, nil }
func (f *fakeDevice) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func newTestManager() (*Manager, *fakeDevice) {
	dev := &fakeDevice{name: "utun9"}
	m := New(dev, func(ifname string) error { return nil })
	return m, dev
}

func TestManagerCreateUp(t *testing.T) {
	m, _ := newTestManager()
	ifname, err := m.Create("10.0.0.1/24", 1420)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ifname == "" {
		t.Fatal("Create: empty ifname")
	}
	state, gotIfname, addr, mtu, _ := m.Status()
	if state != StateUp {
		t.Fatalf("state = %q, want up", state)
	}
	if gotIfname != ifname {
		t.Fatalf("ifname = %q, want %q", gotIfname, ifname)
	}
	if addr != "10.0.0.1/24" {
		t.Fatalf("addr = %q", addr)
	}
	if mtu != 1420 {
		t.Fatalf("mtu = %d", mtu)
	}
}

func TestManagerDuplicateCreate(t *testing.T) {
	m, _ := newTestManager()
	if _, err := m.Create("10.0.0.1/24", 1420); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := m.Create("10.0.0.2/24", 1420); !errors.Is(err, ErrExists) {
		t.Fatalf("second Create err = %v, want ErrExists", err)
	}
}

func TestManagerDeleteDown(t *testing.T) {
	m, _ := newTestManager()
	if _, err := m.Create("10.0.0.1/24", 1420); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := m.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	state, _, _, _, _ := m.Status()
	if state != StateDown {
		t.Fatalf("state = %q, want down", state)
	}
}

func TestManagerDeleteNoTun(t *testing.T) {
	m, _ := newTestManager()
	if err := m.Delete(); !errors.Is(err, ErrNoTun) {
		t.Fatalf("Delete err = %v, want ErrNoTun", err)
	}
}

func TestManagerRingLog(t *testing.T) {
	m, _ := newTestManager()
	for i := 0; i < 250; i++ {
		m.PushLog("rx", "pkt")
	}
	got := m.Logs(300)
	if len(got) != 200 {
		t.Fatalf("len(Logs(300)) = %d, want 200", len(got))
	}
	got10 := m.Logs(10)
	if len(got10) != 10 {
		t.Fatalf("len(Logs(10)) = %d, want 10", len(got10))
	}
}

func TestManagerStats(t *testing.T) {
	m, _ := newTestManager()
	m.AddRx(100)
	m.AddRx(50)
	m.AddTx(200)
	_, _, _, _, s := m.Status()
	if s.RxPackets != 2 || s.RxBytes != 150 {
		t.Fatalf("rx = %+v, want packets=2 bytes=150", s)
	}
	if s.TxPackets != 1 || s.TxBytes != 200 {
		t.Fatalf("tx = %+v, want packets=1 bytes=200", s)
	}
}

func TestManagerDevice(t *testing.T) {
	m, dev := newTestManager()
	if m.Device() != dev {
		t.Fatal("Device() did not return injected device")
	}
}
