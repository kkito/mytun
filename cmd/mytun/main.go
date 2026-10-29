// Command mytun 是建卡 + 读写 IP 包的 demo（需 root）。
//
//	macOS: sudo mytun --addr 10.10.0.1/24
//	Linux: sudo mytun --name mytun0 --addr 10.10.0.1/24
//
// 另开终端 ping tun 地址即可看到抓包；默认 --echo 会回 ICMP echo reply。
package main

import (
	"flag"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"

	"mytun/internal/echo"
	"mytun/internal/netconf"
	"mytun/internal/tun"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	name := flag.String("name", "", "接口名（macOS 建议留空自动分配 utunN；Linux 如 mytun0）")
	addr := flag.String("addr", "10.10.0.1/24", "TUN 地址，CIDR 格式，仅 IPv4")
	mtu := flag.Int("mtu", 1420, "MTU")
	echoOn := flag.Bool("echo", true, "是否自动回 ICMP echo reply")
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

	if err := netconf.ConfigAddr(ifName, prefix); err != nil {
		log.Fatalf("配地址失败: %v", err)
	}
	log.Printf("地址已配置: %s dev %s", prefix, ifName)
	log.Printf("试试: ping %s", prefix.Addr())

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
		handlePacket(t, buf[:n], *echoOn)
	}
}

func handlePacket(t *tun.TUN, pkt []byte, echoOn bool) {
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
	log.Printf("RX ip src=%s dst=%s proto=%d len=%d", src, dst, pkt[9], len(pkt))

	if echoOn && pkt[9] == 1 {
		if reply, ok := echo.ReplyToPing(pkt); ok {
			if _, err := t.Write(reply); err != nil {
				log.Printf("TX echo reply 失败: %v", err)
			} else {
				log.Printf("TX echo reply -> %s", src)
			}
		}
	}
}
