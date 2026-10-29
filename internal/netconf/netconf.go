// Package netconf 按平台给 TUN 接口配地址 + up（需 root）。
package netconf

import (
	"fmt"
	"net/netip"
	"os/exec"
	"runtime"
)

// IsDarwin 报告是否为 macOS。
func IsDarwin() bool { return runtime.GOOS == "darwin" }

// ConfigAddr 给 ifName 配 prefix 地址并 up。
// Linux: ip addr add / ip link set up
// macOS: utun 是点对点接口，用 ifconfig <name> inet <ip> <peer> netmask .. up
func ConfigAddr(ifName string, prefix netip.Prefix) error {
	if IsDarwin() {
		ip := prefix.Addr().String()
		peer := firstHostPeer(prefix, ip)
		c := []string{"ifconfig", ifName, "inet", ip, peer, "netmask", Netmask(prefix.Bits()), "up"}
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %v: %s", c, err, out)
		}
		return nil
	}
	cmds := [][]string{
		{"ip", "addr", "add", prefix.String(), "dev", ifName},
		{"ip", "link", "set", "dev", ifName, "up"},
	}
	for _, c := range cmds {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %v: %s", c, err, out)
		}
	}
	return nil
}

// Netmask 把 CIDR bits 转成点分十进制，供 macOS ifconfig 用。
func Netmask(bits int) string {
	var b [4]byte
	for i := 0; i < 32; i++ {
		if i < bits {
			b[i/8] |= 1 << (7 - uint(i%8))
		}
	}
	return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
}

// firstHostPeer 取网段内一个与 ip 不同的主机地址做 utun 对端占位。
func firstHostPeer(prefix netip.Prefix, ip string) string {
	base := prefix.Masked().Addr().As4()
	for i := 1; i < 254; i++ {
		c := fmt.Sprintf("%d.%d.%d.%d", base[0], base[1], base[2], base[3]+byte(i))
		if c != ip {
			return c
		}
	}
	return ip
}
