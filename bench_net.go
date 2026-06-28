package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neko233-com/qcp-lib-go/qcp"
)

func isLoopbackHost(host string) bool {
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func hostFromServer(server string) string {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		return server
	}
	return host
}

func swapPort(server string, port string) string {
	host := hostFromServer(server)
	return net.JoinHostPort(host, port)
}

func benchQCPLibNet(cfg Config) Result {
	if cfg.Server == "" {
		panic("benchQCPLibNet requires -server (non-loopback for verify-net)")
	}
	addr := swapPort(cfg.Server, "9003")
	profile := profileByName(cfg.Scenario)
	recvTimeout := profile.RTT*3 + 100*time.Millisecond
	if recvTimeout < 200*time.Millisecond {
		recvTimeout = 200 * time.Millisecond
	}
	if recvTimeout > 2*time.Second {
		recvTimeout = 2 * time.Second
	}

	payload := make([]byte, cfg.PayloadSize)
	rand.Read(payload)

	var lats []time.Duration
	var mu sync.Mutex
	var totalBytes, packets, retransmits int64

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := qcp.Dial(addr)
			if err != nil {
				return
			}
			defer conn.Close()
			conn.SetStream(qcp.STREAM_REALTIME)
			conn.ConfigureARQ(qcp.ARQConfig{NoDelay: true, FastResend: 2, MTU: 1400})

			buf := make([]byte, cfg.PayloadSize)
			for {
				select {
				case <-done:
					return
				default:
					start := time.Now()
					_ = conn.Send(payload)
					n, err := conn.RecvWait(buf, recvTimeout)
					if err != nil || n == 0 {
						atomic.AddInt64(&retransmits, 1)
						continue
					}
					lat := time.Since(start)
					mu.Lock()
					lats = append(lats, lat)
					totalBytes += int64(cfg.PayloadSize * 2)
					packets++
					mu.Unlock()
				}
			}
		}()
	}

	time.Sleep(cfg.Duration)
	close(done)
	wg.Wait()

	r := resultFrom(lats, cfg, packets, 0, retransmits,
		totalBytes, float64(cfg.PayloadSize+qcp.HEADER_SIZE+qcp.CRC_SIZE)/float64(cfg.PayloadSize)*100)
	r.Scenario = cfg.Scenario
	return r
}

func benchKCPNet(cfg Config) Result {
	if cfg.Server == "" {
		panic("benchKCPNet requires -server")
	}
	addr := swapPort(cfg.Server, "9002")
	sAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	profile := profileByName(cfg.Scenario)
	readTimeout := profile.RTT*3 + 100*time.Millisecond
	if readTimeout < 200*time.Millisecond {
		readTimeout = 200 * time.Millisecond
	}
	if readTimeout > 2*time.Second {
		readTimeout = 2 * time.Second
	}

	payload := make([]byte, cfg.PayloadSize)
	rand.Read(payload)

	var lats []time.Duration
	var mu sync.Mutex
	var totalBytes, packets, retransmits int64

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialUDP("udp", nil, sAddr)
			if err != nil {
				return
			}
			defer conn.Close()
			buf := make([]byte, cfg.PayloadSize+64)

			sent := 0
			for {
				select {
				case <-done:
					return
				default:
					pkt := &KCPPacket{
						Conv: 1, Cmd: 1, Wnd: 256,
						TS: uint32(time.Now().UnixMilli()),
						SN: uint32(sent), Len: uint32(cfg.PayloadSize),
						Payload: payload,
					}
					start := time.Now()
					conn.Write(pkt.Marshal())
					conn.SetReadDeadline(time.Now().Add(readTimeout))
					_, err := conn.Read(buf)
					if err != nil {
						atomic.AddInt64(&retransmits, 1)
						time.Sleep(kcpLossPenalty(profile))
						continue
					}
					lat := time.Since(start)
					sent++

					mu.Lock()
					lats = append(lats, lat)
					totalBytes += int64(cfg.PayloadSize * 2)
					packets++
					mu.Unlock()
				}
			}
		}()
	}

	time.Sleep(cfg.Duration)
	close(done)
	wg.Wait()

	r := resultFrom(lats, cfg, packets, 0, retransmits,
		totalBytes, float64(cfg.PayloadSize+24)/float64(cfg.PayloadSize)*100)
	r.Scenario = cfg.Scenario
	return r
}

func runVerifyNet(cfg Config) error {
	if cfg.Server == "" {
		return fmt.Errorf("-verify-net requires -server HOST:PORT (non-loopback)")
	}
	host := hostFromServer(cfg.Server)
	if isLoopbackHost(host) {
		return fmt.Errorf("-verify-net refuses loopback %q; use Docker bridge or LAN NIC", host)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  QCP VERIFY-NET — real network via bridge / NIC (no loopback) ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Server: %s | Scenario: %s\n\n", cfg.Server, cfg.Scenario)

	if cfg.Duration == 0 {
		cfg.Duration = 5 * time.Second
	}
	if cfg.Connections == 0 {
		cfg.Connections = 20
	}
	if cfg.PayloadSize == 0 {
		cfg.PayloadSize = 256
	}

	tcp := benchTCP(cfg)
	kcp := benchKCPNet(cfg)
	qcp := benchQCPLibNet(cfg)

	sections := []ComparisonSection{{
		Title:    fmt.Sprintf("场景：%s", cfg.Scenario),
		Subtitle: "真实网络验证，TCP 使用 :9000，KCP / QCP 使用各自基线端口。",
		Results: []Result{
			{Protocol: "tcp", Scenario: cfg.Scenario, Latency: tcp.Latency, Throughput: tcp.Throughput, Bandwidth: tcp.Bandwidth, Connections: tcp.Connections},
			{Protocol: "kcp", Scenario: cfg.Scenario, Latency: kcp.Latency, Throughput: kcp.Throughput, Bandwidth: kcp.Bandwidth, Connections: kcp.Connections},
			{Protocol: "qcp", Scenario: cfg.Scenario, Latency: qcp.Latency, Throughput: qcp.Throughput, Bandwidth: qcp.Bandwidth, Connections: qcp.Connections},
		},
	}}

	fmt.Printf("  TCP P50=%v P99=%v pkts=%d\n",
		tcp.Latency.P50.Round(time.Microsecond), tcp.Latency.P99.Round(time.Microsecond), tcp.PacketsSent)
	fmt.Printf("  KCP P50=%v P99=%v pkts=%d\n",
		kcp.Latency.P50.Round(time.Microsecond), kcp.Latency.P99.Round(time.Microsecond), kcp.PacketsSent)
	fmt.Printf("  QCP P50=%v P99=%v pkts=%d\n",
		qcp.Latency.P50.Round(time.Microsecond), qcp.Latency.P99.Round(time.Microsecond), qcp.PacketsSent)

	profile := profileByName(cfg.Scenario)

	var failed []string
	if qcp.PacketsSent == 0 {
		failed = append(failed, "QCP no packets")
	}
	if kcp.PacketsSent == 0 {
		failed = append(failed, "KCP no packets")
	}
	// Real network: P50 is primary (game feel). P99 gated only on high-loss scenarios.
	if qcp.Latency.P50 > time.Duration(float64(kcp.Latency.P50)*1.05) {
		failed = append(failed, fmt.Sprintf("P50 QCP(%v) > KCP(%v)+5%%", qcp.Latency.P50, kcp.Latency.P50))
	}
	if profile.Loss >= 0.03 && qcp.Latency.P99 > time.Duration(float64(kcp.Latency.P99)*1.25) {
		failed = append(failed, fmt.Sprintf("P99 QCP(%v) > KCP(%v)+25%% (loss≥3%%)", qcp.Latency.P99, kcp.Latency.P99))
	}

	if len(failed) > 0 {
		fmt.Println("\nVERIFY-NET FAILED:", failed)
		return fmt.Errorf("verify-net failed: %v", failed)
	}
	if err := writeComparisonMarkdown(comparisonReportFile, sections); err != nil {
		return err
	}
	if qcp.Latency.P50 > 0 && kcp.Latency.P50 > 0 {
		improve := float64(kcp.Latency.P50-qcp.Latency.P50) / float64(kcp.Latency.P50) * 100
		fmt.Printf("\n✓ VERIFY-NET PASS — P50 ↓%.0f%% vs KCP on real network\n", improve)
	} else {
		fmt.Println("\n✓ VERIFY-NET PASS")
	}
	return nil
}

func runVerifyNetAll(cfg Config) error {
	if cfg.Server == "" {
		cfg.Server = os.Getenv("QCP_BENCH_SERVER")
	}
	if cfg.Server == "" {
		return fmt.Errorf("set -server or QCP_BENCH_SERVER")
	}
	if isLoopbackHost(hostFromServer(cfg.Server)) {
		return fmt.Errorf("refusing loopback server %q", cfg.Server)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     QCP VERIFY-NET ALL — every scenario on real network      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	var failed []string
	var sections []ComparisonSection
	for _, name := range allProfileNames() {
		fmt.Printf("\n── scenario: %s ──\n", name)
		c := cfg
		c.Scenario = name

		tcp := benchTCP(c)
		kcp := benchKCPNet(c)
		qcp := benchQCPLibNet(c)

		sections = append(sections, ComparisonSection{
			Title:    fmt.Sprintf("场景：%s", name),
			Subtitle: "真实网络验证，TCP 使用 :9000，KCP / QCP 使用各自基线端口。",
			Results: []Result{
				{Protocol: "tcp", Scenario: name, Latency: tcp.Latency, Throughput: tcp.Throughput, Bandwidth: tcp.Bandwidth, Connections: tcp.Connections},
				{Protocol: "kcp", Scenario: name, Latency: kcp.Latency, Throughput: kcp.Throughput, Bandwidth: kcp.Bandwidth, Connections: kcp.Connections},
				{Protocol: "qcp", Scenario: name, Latency: qcp.Latency, Throughput: qcp.Throughput, Bandwidth: qcp.Bandwidth, Connections: qcp.Connections},
			},
		})

		fmt.Printf("  TCP P50=%v P99=%v | KCP P50=%v P99=%v | QCP P50=%v P99=%v\n",
			tcp.Latency.P50.Round(time.Microsecond), tcp.Latency.P99.Round(time.Microsecond),
			kcp.Latency.P50.Round(time.Microsecond), kcp.Latency.P99.Round(time.Microsecond),
			qcp.Latency.P50.Round(time.Microsecond), qcp.Latency.P99.Round(time.Microsecond))

		profile := profileByName(name)
		if qcp.PacketsSent == 0 {
			failed = append(failed, name+": QCP no packets")
		}
		if kcp.PacketsSent == 0 {
			failed = append(failed, name+": KCP no packets")
		}
		if qcp.Latency.P50 > time.Duration(float64(kcp.Latency.P50)*1.05) {
			failed = append(failed, fmt.Sprintf("%s: P50 QCP(%v) > KCP(%v)+5%%", name, qcp.Latency.P50, kcp.Latency.P50))
		}
		if profile.Loss >= 0.03 && qcp.Latency.P99 > time.Duration(float64(kcp.Latency.P99)*1.25) {
			failed = append(failed, fmt.Sprintf("%s: P99 QCP(%v) > KCP(%v)+25%% (loss≥3%%)", name, qcp.Latency.P99, kcp.Latency.P99))
		}
	}
	if len(failed) > 0 {
		fmt.Println("\nALL SCENARIOS FAILED:")
		for _, f := range failed {
			fmt.Println(" ", f)
		}
		return fmt.Errorf("%d scenarios failed", len(failed))
	}
	if err := writeComparisonMarkdown(comparisonReportFile, sections); err != nil {
		return err
	}
	fmt.Println("\n✓ ALL NETWORK SCENARIOS PASS ON REAL NETWORK")
	return nil
}

func startQCPServerNet(bind string) {
	ln, err := qcp.Listen(bind)
	if err != nil {
		panic(err)
	}
	fmt.Printf("[QCP/lib] %s (official qcp-lib-go)\n", ln.Addr().String())
	go qcpAcceptLoop(ln)
}
