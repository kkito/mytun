package echo

import (
	"encoding/binary"
	"testing"
)

func TestChecksum(t *testing.T) {
	// 已知：全 0 的 IP 头校验和应为 0xffff
	hdr := make([]byte, 20)
	if got := Checksum(hdr); got != 0xffff {
		t.Fatalf("zero header checksum = %#04x, want 0xffff", got)
	}
}

func TestReplyToPing(t *testing.T) {
	// 构造最小 ICMP echo request：IP 头(20) + ICMP(8)
	pkt := make([]byte, 28)
	pkt[0] = 0x45
	binary.BigEndian.PutUint16(pkt[2:4], 28)
	pkt[8] = 64
	pkt[9] = 1 // ICMP
	copy(pkt[12:16], []byte{1, 2, 3, 4})
	copy(pkt[16:20], []byte{10, 10, 0, 1})
	binary.BigEndian.PutUint16(pkt[10:12], Checksum(pkt[:20]))
	pkt[20] = 8 // echo request
	binary.BigEndian.PutUint16(pkt[22:24], Checksum(pkt[20:]))

	reply, ok := ReplyToPing(pkt)
	if !ok {
		t.Fatal("ReplyToPing returned not-ok for echo request")
	}
	if reply[20] != 0 {
		t.Fatalf("icmp type = %d, want 0", reply[20])
	}
	// src/dst 应交换
	if reply[12] != 10 || reply[16] != 1 {
		t.Fatalf("src/dst not swapped: %v", reply[12:20])
	}
	// 校验和自洽：对含校验和的报文再算应为 0
	if Checksum(reply[:20]) != 0 {
		t.Fatalf("ip checksum invalid: %#04x", Checksum(reply[:20]))
	}
	if Checksum(reply[20:]) != 0 {
		t.Fatalf("icmp checksum invalid: %#04x", Checksum(reply[20:]))
	}
}

func TestReplyToPingNonEcho(t *testing.T) {
	pkt := make([]byte, 28)
	pkt[0] = 0x45
	pkt[9] = 6 // TCP
	if _, ok := ReplyToPing(pkt); ok {
		t.Fatal("expected not-ok for TCP packet")
	}
}
