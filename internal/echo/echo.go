// Package echo 提供 ICMP echo reply 构造（用于验证 TUN Write 通路）。
package echo

import "encoding/binary"

// ReplyToPing 若 pkt 是 ICMP echo request，返回 echo reply 报文；否则 ok=false。
// 做法：拷贝整包、交换 IP src/dst、ICMP type 8->0，重算 IP 与 ICMP 校验和。
func ReplyToPing(pkt []byte) (out []byte, ok bool) {
	if len(pkt) < 20 || pkt[0]>>4 != 4 || pkt[9] != 1 {
		return nil, false
	}
	ihl := int(pkt[0]&0x0f) * 4
	if len(pkt) < ihl+8 {
		return nil, false
	}
	if pkt[ihl] != 8 { // echo request
		return nil, false
	}
	out = make([]byte, len(pkt))
	copy(out, pkt)
	copy(out[12:16], pkt[16:20])
	copy(out[16:20], pkt[12:16])
	out[10], out[11] = 0, 0
	binary.BigEndian.PutUint16(out[10:12], Checksum(out[:ihl]))
	oicmp := out[ihl:]
	oicmp[0] = 0
	oicmp[2], oicmp[3] = 0, 0
	binary.BigEndian.PutUint16(oicmp[2:4], Checksum(oicmp))
	return out, true
}

// Checksum 计算 RFC 1071 校验和。
func Checksum(b []byte) uint16 {
	var s uint32
	for i := 0; i+1 < len(b); i += 2 {
		s += uint32(binary.BigEndian.Uint16(b[i:]))
	}
	if len(b)%2 == 1 {
		s += uint32(b[len(b)-1]) << 8
	}
	for s>>16 > 0 {
		s = (s & 0xffff) + (s >> 16)
	}
	return ^uint16(s)
}
