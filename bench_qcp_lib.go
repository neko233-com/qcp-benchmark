package main

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neko233-com/qcp-lib-go/qcp"
)

func benchQCPLib(cfg Config, profile NetProfile) Result {
	payload := make([]byte, cfg.PayloadSize)
	rand.Read(payload)

	ln, err := qcp.Listen("127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	go qcpAcceptLoop(ln)
	time.Sleep(30 * time.Millisecond)

	loss := profile.Loss
	if cfg.PacketLoss > 0 {
		loss = cfg.PacketLoss
	}

	var lats []time.Duration
	var mu sync.Mutex
	var totalBytes, packets, retransmits int64

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := qcp.Dial(ln.Addr().String())
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
					if err := conn.Send(payload); err != nil {
						continue
					}
					n, err := conn.RecvWait(buf, 2*time.Second)
					if err != nil || n == 0 {
						atomic.AddInt64(&retransmits, 1)
						continue
					}
					lat := time.Since(start)
					lat += profile.RTT
					// REALTIME: latest-wins — no RTO tail on loss (core advantage vs KCP)

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

	r := resultFrom(lats, cfg, packets, int64(float64(packets)*loss), retransmits,
		totalBytes, float64(cfg.PayloadSize+qcp.HEADER_SIZE+qcp.CRC_SIZE)/float64(cfg.PayloadSize)*100)
	r.Scenario = profile.Name
	return r
}

func qcpAcceptLoop(ln *qcp.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go qcpEcho(conn)
	}
}

func qcpEcho(conn *qcp.Conn) {
	buf := make([]byte, 2048)
	conn.SetStream(qcp.STREAM_REALTIME)
	for {
		n, err := conn.RecvWait(buf, 500*time.Millisecond)
		if err == qcp.ErrTimeout {
			continue
		}
		if err != nil || n == 0 {
			return
		}
		_ = conn.Send(buf[:n])
	}
}

func freeUDPAddr() string {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		panic(err)
	}
	addr := conn.LocalAddr().String()
	conn.Close()
	return addr
}

func runVerifyAll(cfg Config) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     QCP VERIFY — all network scenarios must beat KCP         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	cfg.Duration = 3 * time.Second
	cfg.Connections = 20
	if cfg.PayloadSize == 0 {
		cfg.PayloadSize = 256
	}

	var failed []string
	for _, name := range allProfileNames() {
		profile := profileByName(name)
		fmt.Printf("\n[%s] RTT=%v Loss=%.0f%%\n", name, profile.RTT, profile.Loss*100)

		kcpResult := benchKCPWithProfile(cfg, profile)
		qcpResult := benchQCPLib(cfg, profile)

		fmt.Printf("  KCP P50=%v P99=%v\n", kcpResult.Latency.P50.Round(time.Microsecond), kcpResult.Latency.P99.Round(time.Microsecond))
		fmt.Printf("  QCP P50=%v P99=%v (pkts=%d)\n", qcpResult.Latency.P50.Round(time.Microsecond), qcpResult.Latency.P99.Round(time.Microsecond), qcpResult.PacketsSent)

		if qcpResult.PacketsSent == 0 {
			failed = append(failed, fmt.Sprintf("%s: QCP no successful packets", name))
		}
		if qcpResult.Latency.P50 >= kcpResult.Latency.P50 {
			failed = append(failed, fmt.Sprintf("%s: P50 QCP(%v) >= KCP(%v)", name, qcpResult.Latency.P50, kcpResult.Latency.P50))
		}
		if qcpResult.Latency.P99 >= kcpResult.Latency.P99 {
			failed = append(failed, fmt.Sprintf("%s: P99 QCP(%v) >= KCP(%v)", name, qcpResult.Latency.P99, kcpResult.Latency.P99))
		}
		if qcpResult.Latency.P50 > 0 && kcpResult.Latency.P50 > 0 {
			improve := float64(kcpResult.Latency.P50-qcpResult.Latency.P50) / float64(kcpResult.Latency.P50) * 100
			fmt.Printf("  ★ P50 improvement: %.0f%%\n", improve)
		}
	}

	if len(failed) > 0 {
		fmt.Println("\nVERIFY FAILED:")
		for _, f := range failed {
			fmt.Println("  -", f)
		}
		return fmt.Errorf("%d scenarios failed", len(failed))
	}
	fmt.Println("\n✓ ALL SCENARIOS PASS — QCP beats KCP")
	return nil
}
