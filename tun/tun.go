package tun

import (
	"fmt"

	wgTun "golang.zx2c4.com/wireguard/tun"
)

// TUN 统一接口：Read/Write 都是纯 IP 包（已剥掉各平台的附加包头）。
type TUN struct {
	dev wgTun.Device
}

func Create(name string, mtu int) (*TUN, error) {
	dev, err := wgTun.CreateTUN(name, mtu)
	if err != nil {
		return nil, fmt.Errorf("create tun %q: %w", name, err)
	}
	// wireguard-go 的 fd 消费模型是事件驱动的，但 Device 本身
	// 也提供 file-based Read/Write（Linux/Darwin 都有实现）。
	return &TUN{dev: dev}, nil
}

// ActualName 返回内核实际分配到的接口名（传 "utun" 会自动分配 utunN）。
func (t *TUN) ActualName() (string, error) { return t.dev.Name() }

func (t *TUN) Read(p []byte) (int, error) {
	// wireguard-go 新版是批量 API：Read(bufs, sizes, offset)，返回包数。
	// Darwin/Linux 的实现都已处理好包头，这里只取 1 个包。
	bufs := [][]byte{p}
	sizes := []int{0}
	n, err := t.dev.Read(bufs, sizes, 0)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, fmt.Errorf("read: no packet")
	}
	return sizes[0], nil
}

func (t *TUN) Write(p []byte) (int, error) { return t.dev.Write([][]byte{p}, 0) }

func (t *TUN) Close() error { return t.dev.Close() }
