package packet_test

import (
	"encoding/binary"
	"net/netip"
	"testing"

	"mytun/internal/echo"
	"mytun/internal/packet"
)

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	a, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestBuildIPv4EchoRequestHeader(t *testing.T) {
	src := mustAddr(t, "10.0.0.1")
	dst := mustAddr(t, "10.0.0.2")
	pkt, err := packet.BuildIPv4EchoRequest(src, dst, 64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkt) != 64 {
		t.Fatalf("len = %d, want 64", len(pkt))
	}
	if ver := pkt[0] >> 4; ver != 4 {
		t.Fatalf("ver = %d, want 4", ver)
	}
	if proto := pkt[9]; proto != 1 {
		t.Fatalf("proto = %d, want 1", proto)
	}
	if typ := pkt[20]; typ != 8 {
		t.Fatalf("icmp type = %d, want 8", typ)
	}
	if got := netip.AddrFrom4([4]byte(pkt[12:16])); got != src {
		t.Fatalf("src = %v, want %v", got, src)
	}
	if got := netip.AddrFrom4([4]byte(pkt[16:20])); got != dst {
		t.Fatalf("dst = %v, want %v", got, dst)
	}
	if total := int(binary.BigEndian.Uint16(pkt[2:4])); total != 64 {
		t.Fatalf("total len field = %d, want 64", total)
	}
	if got := echo.Checksum(pkt[:20]); got != 0 {
		t.Fatalf("ip checksum not self-consistent: %#04x", got)
	}
	if got := echo.Checksum(pkt[20:]); got != 0 {
		t.Fatalf("icmp checksum not self-consistent: %#04x", got)
	}
}

func TestBuildIPv4EchoRequestBadSize(t *testing.T) {
	src := mustAddr(t, "10.0.0.1")
	dst := mustAddr(t, "10.0.0.2")
	for _, size := range []int{0, 20, 27, 65536, -1} {
		if _, err := packet.BuildIPv4EchoRequest(src, dst, size); err == nil {
			t.Fatalf("size %d: expected error, got nil", size)
		}
	}
}

func TestParseSummary(t *testing.T) {
	src := mustAddr(t, "10.0.0.1")
	dst := mustAddr(t, "10.0.0.2")

	normal, err := packet.BuildIPv4EchoRequest(src, dst, 64)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := packet.ParseSummary(normal), "10.0.0.1->10.0.0.2 proto=1 len=64"; got != want {
		t.Fatalf("normal: got %q want %q", got, want)
	}

	short := []byte{0x45, 0x00, 0x00}
	if got, want := packet.ParseSummary(short), "short len=3"; got != want {
		t.Fatalf("short: got %q want %q", got, want)
	}

	tcp := make([]byte, 40)
	tcp[0] = 0x45
	tcp[9] = 6
	copy(tcp[12:16], []byte{10, 0, 0, 1})
	copy(tcp[16:20], []byte{10, 0, 0, 2})
	if got, want := packet.ParseSummary(tcp), "10.0.0.1->10.0.0.2 proto=6 len=40"; got != want {
		t.Fatalf("tcp: got %q want %q", got, want)
	}

	v6 := make([]byte, 40)
	v6[0] = 0x60
	if got, want := packet.ParseSummary(v6), "non-ipv4 ver=6"; got != want {
		t.Fatalf("v6: got %q want %q", got, want)
	}
}
