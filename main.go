// Package main 是 mytun 的 demo CLI：建卡 + 配地址 + 读写 IP 包。
//
// 用法（都需要 root）：
//
//	macOS: sudo go run . --addr 10.10.0.1/24 --echo
//	Linux: sudo go run . --name mytun0 --addr 10.10.0.1/24 --echo
//
// 另开终端 ping 即可看到抓包：ping 10.10.0.1
// --echo 会对 ping 做 ICMP echo reply，证明 Write 通路是好的。
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"mytun/tun"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	name := flag.String("name", "", "接口名（macOS 建议留空自动分配 utunN；Linux 如 mytun0）")
	addr := flag.String("addr", "10.10.0.1/24", "TUN 地址，CIDR 格式，仅 IPv4")
	mtu := flag.Int("mtu", 1420, "MTU")
	echo := flag.Bool("echo", true, "是否自动回 ICMP echo reply（ping 验证 Write 通路）")
	flag.Parse()

	prefix, err := netip.ParsePrefix(*addr)
	if err != nil || !prefix.Addr().Is4() {
		log.Fatalf("addr 必须是 IPv4 CIDR，如 10.10.0.1/24: %v", err)
	}

	t, err := tun.Create(*name, *mtu)
	if err != nil {
		log.Fatalf("建卡失败（记得 sudo）: %v", err)
	}
	defer t.Close()

	ifName, err := t.ActualName()
	if err != nil {
		log.Fatalf("取接口名失败: %v", err)
	}
	log.Printf("TUN 已创建: %s mtu=%d", ifName, *mtu)

	if err := configAddr(ifName, prefix); err != nil {
		log.Fatalf("配地址失败: %v", err)
	}
	log.Printf("地址已配置: %s dev %s", prefix, ifName)
	log.Printf("试试: ping %s", prefix.Addr())

	// 优雅退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Printf("退出，接口 %s 关闭", ifName)
		t.Close()
		os.Exit(0)
	}()

	buf := make([]byte, 65535)
	for {
		n, err := t.Read(buf)
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		pkt := buf[:n]
		handlePacket(t, pkt, *echo)
	}
}

// handlePacket 解析 IPv4 头并打印，echo 模式下回 ping。
func handlePacket(t *tun.TUN, pkt []byte, echo bool) {
	if len(pkt) < 20 {
		log.Printf("短包 len=%d", len(pkt))
		return
	}
	if pkt[0]>>4 != 4 {
		log.Printf("非 IPv4 包 ver=%d len=%d", pkt[0]>>4, len(pkt))
		return
	}
	src := net.IP(pkt[12:16])
	dst := net.IP(pkt[16:20])
	proto := pkt[9]
	totalLen := int(binary.BigEndian.Uint16(pkt[2:4]))
	log.Printf("RX ip src=%s dst=%s proto=%d len=%d", src, dst, proto, totalLen)

	if echo && proto == 1 { // ICMP
		reply, ok := icmpEchoReply(pkt)
		if ok {
			if _, err := t.Write(reply); err != nil {
				log.Printf("TX echo reply 失败: %v", err)
			} else {
				log.Printf("TX echo reply -> %s", src)
			}
		}
	}
}

// icmpEchoReply 把 echo request 原地改成 echo reply：交换 IP、ICMP type 8->0，重算校验和。
func icmpEchoReply(pkt []byte) ([]byte, bool) {
	ihl := int(pkt[0]&0x0f) * 4
	if len(pkt) < ihl+8 || pkt[9] != 1 {
		return nil, false
	}
	icmp := pkt[ihl:]
	if icmp[0] != 8 { // echo request
		return nil, false
	}
	out := make([]byte, len(pkt))
	copy(out, pkt)
	// 交换 src/dst
	copy(out[12:16], pkt[16:20])
	copy(out[16:20], pkt[12:16])
	// IP 校验和清零重算
	out[10], out[11] = 0, 0
	sum := checksum(out[:ihl])
	binary.BigEndian.PutUint16(out[10:12], sum)
	// ICMP type 8->0，校验和重算
	oicmp := out[ihl:]
	oicmp[0] = 0
	oicmp[2], oicmp[3] = 0, 0
	sum = checksum(oicmp)
	binary.BigEndian.PutUint16(oicmp[2:4], sum)
	return out, true
}

func checksum(b []byte) uint16 {
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

// configAddr 按平台配地址 + up 接口。
func configAddr(ifName string, prefix netip.Prefix) error {
	ip := prefix.Addr().String()
	if isDarwin() {
		// utun 是点对点接口：ifconfig <name> inet <ip> <peer> up
		// peer 取网段内另一个地址即可（demo 用网关 .1 对端就用 .2 占位无妨，
		// 实际收包走的是目的路由，这里只是让接口 up）。
		peer := firstHostPeer(prefix, ip)
		cmds := [][]string{
			{"ifconfig", ifName, "inet", ip, peer, "netmask", netmask(prefix.Bits()), "up"},
		}
		for _, c := range cmds {
			if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
				return fmt.Errorf("%v: %v: %s", c, err, out)
			}
		}
		return nil
	}
	// Linux
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
