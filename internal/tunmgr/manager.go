package tunmgr

import (
	"errors"
	"sync"
	"time"
)

// State is "down" | "up".
type State string

const (
	StateDown State = "down"
	StateUp   State = "up"
)

// Stats counts packets and bytes in both directions.
type Stats struct {
	RxPackets uint64
	TxPackets uint64
	RxBytes   uint64
	TxBytes   uint64
}

// LogEntry is one ring-log record.
type LogEntry struct {
	Time    string
	Dir     string
	Summary string
}

// Device 是 TUN 设备的最小接口：fake device 实现此接口注入，单测零 root。
type Device interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	ActualName() (string, error)
	Close() error
}

var (
	ErrExists = errors.New("tun already exists")
	ErrNoTun  = errors.New("no tun")
)

const maxLogs = 200

// Manager 是 mutex 单实例状态机 + stats + 环形 log。
type Manager struct {
	mu         sync.Mutex
	dev        Device
	configurer func(ifname string) error
	up         bool
	ifname     string
	addr       string
	mtu        int
	stats      Stats
	logs       []LogEntry
}

// New 创建下线状态的 Manager。dev 为底层 TUN 设备（测试注入 fake），
// configurer 负责配地址（生产环境传入 netconf.ConfigAddr 包装）。
func New(dev Device, configurer func(ifname string) error) *Manager {
	return &Manager{dev: dev, configurer: configurer}
}

// Device 暴露底层 device，供后续 send 路径使用。
func (m *Manager) Device() Device { return m.dev }

// Create 建卡并配地址；已存在返回 ErrExists。
func (m *Manager) Create(addr string, mtu int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.up {
		return "", ErrExists
	}
	ifname, err := m.dev.ActualName()
	if err != nil {
		return "", err
	}
	if m.configurer != nil {
		if err := m.configurer(ifname); err != nil {
			return "", err
		}
	}
	m.up = true
	m.ifname = ifname
	m.addr = addr
	m.mtu = mtu
	return ifname, nil
}

// Delete 删卡；不存在返回 ErrNoTun。
func (m *Manager) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.up {
		return ErrNoTun
	}
	if err := m.dev.Close(); err != nil {
		return err
	}
	m.up = false
	m.ifname = ""
	m.addr = ""
	m.mtu = 0
	return nil
}

// Status 返回当前状态快照。
func (m *Manager) Status() (State, string, string, int, Stats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.up {
		return StateDown, "", "", 0, m.stats
	}
	return StateUp, m.ifname, m.addr, m.mtu, m.stats
}

// PushLog 追加一条日志，超 200 条从头截断。
func (m *Manager) PushLog(dir, summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Time: time.Now().Format(time.RFC3339), Dir: dir, Summary: summary})
	if len(m.logs) > maxLogs {
		m.logs = append([]LogEntry(nil), m.logs[len(m.logs)-maxLogs:]...)
	}
}

// Logs 返回最近 limit 条（limit<=0 或超长则返回全部）。
func (m *Manager) Logs(limit int) []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.logs) {
		limit = len(m.logs)
	}
	out := make([]LogEntry, limit)
	copy(out, m.logs[len(m.logs)-limit:])
	return out
}

// AddRx 记录收包 n 字节。
func (m *Manager) AddRx(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.RxPackets++
	m.stats.RxBytes += uint64(n)
}

// AddTx 记录发包 n 字节。
func (m *Manager) AddTx(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TxPackets++
	m.stats.TxBytes += uint64(n)
}
