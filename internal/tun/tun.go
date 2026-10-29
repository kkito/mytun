package tun

import (
	"fmt"
	"runtime"

	wgTun "golang.zx2c4.com/wireguard/tun"
)

// TUN 统一接口：Read/Write 都是纯 IP 包（已剥掉各平台的附加包头）。
// Linux 走 /dev/net/tun，macOS 走 utun（wireguard-go 已处理 utun 4 字节 AF 头）。
type TUN struct {
	dev wgTun.Device
}

func Create(name string, mtu int) (*TUN, error) {
	if name == "" && runtime.GOOS == "darwin" {
		// macOS wireguard-go 只接受 "utun"（自动分配 utunN）或 "utunN"。
		name = "utun"
	}
	dev, err := wgTun.CreateTUN(name, mtu)
	if err != nil {
		return nil, fmt.Errorf("create tun %q: %w", name, err)
	}
	return &TUN{dev: dev}, nil
}

// ActualName 返回内核实际分配到的接口名（macOS 传 "utun" 会自动分配 utunN）。
func (t *TUN) ActualName() (string, error) { return t.dev.Name() }

func (t *TUN) Read(p []byte) (int, error) {
	// wireguard-go 新版是批量 API：Read(bufs, sizes, offset)，返回包数。
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
