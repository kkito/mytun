package main

import (
	"fmt"
	"net/netip"
	"runtime"
)

func isDarwin() bool { return runtime.GOOS == "darwin" }

// netmask 把 CIDR bits 转成点分十进制，供 macOS ifconfig 用。
func netmask(bits int) string {
	m := netip.PrefixFrom(netip.AddrFrom4([4]byte{0, 0, 0, 0}), bits).Masked()
	_ = m
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
