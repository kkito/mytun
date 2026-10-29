// Package packet 提供无状态纯函数：构造可直接 Write 进 TUN 的
// IPv4 ICMP echo request 包，以及给日志用的包摘要。
package packet

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"mytun/internal/echo"
)

// BuildIPv4EchoRequest 构造总长度为 size（含 20 字节 IP 头）的
// IPv4 ICMP echo request，payload 零填充。size 范围 28~65535。
func BuildIPv4EchoRequest(src, dst netip.Addr, size int) ([]byte, error) {
	if size < 28 || size > 65535 {
		return nil, fmt.Errorf("packet: bad size %d, want 28~65535", size)
	}
	if !src.Is4() || !dst.Is4() {
		return nil, fmt.Errorf("packet: src and dst must be IPv4")
	}
	pkt := make([]byte, size)
	pkt[0] = 0x45 // ver=4, IHL=5
	binary.BigEndian.PutUint16(pkt[2:4], uint16(size))
	pkt[8] = 64 // TTL
	pkt[9] = 1  // ICMP
	copy(pkt[12:16], src.AsSlice())
	copy(pkt[16:20], dst.AsSlice())
	binary.BigEndian.PutUint16(pkt[10:12], echo.Checksum(pkt[:20]))
	pkt[20] = 8 // echo request
	binary.BigEndian.PutUint16(pkt[22:24], echo.Checksum(pkt[20:]))
	return pkt, nil
}

// ParseSummary 返回 "src->dst proto=N len=N"；短包返回
// "short len=N"，非 IPv4 返回 "non-ipv4 ver=N"。
func ParseSummary(pkt []byte) string {
	if len(pkt) < 20 {
		return fmt.Sprintf("short len=%d", len(pkt))
	}
	if ver := pkt[0] >> 4; ver != 4 {
		return fmt.Sprintf("non-ipv4 ver=%d", ver)
	}
	return fmt.Sprintf("%d.%d.%d.%d->%d.%d.%d.%d proto=%d len=%d",
		pkt[12], pkt[13], pkt[14], pkt[15],
		pkt[16], pkt[17], pkt[18], pkt[19],
		pkt[9], len(pkt))
}
