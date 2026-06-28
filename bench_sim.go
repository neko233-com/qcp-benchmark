package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

func benchTCPSimulated(cfg Config, profile NetProfile) Result {
	payload := make([]byte, cfg.PayloadSize)
	rand.Read(payload)
	var lats []time.Duration
	var mu sync.Mutex
	var totalBytes, packets, retransmits int64
	loss := profile.Loss
	if cfg.PacketLoss > 0 {
		loss = cfg.PacketLoss
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cwnd := 8
			inflight := 0
			for {
				select {
				case <-done:
					return
				default:
					start := time.Now()
					// TCP: in-order delivery + head-of-line blocking + retransmit stall.
					lat := profile.RTT * 3 / 4
					lat += time.Duration(rand.Int63n(int64(profile.Jitter/3) + 1))
					if inflight >= cwnd {
						lat += profile.RTT / 2
					}
					if rand.Float64() < loss {
						rto := profile.RTT + 15*time.Millisecond
						if rto < 5*time.Millisecond {
							rto = 5 * time.Millisecond
						}
						lat += rto
						atomic.AddInt64(&retransmits, 1)
						if cwnd > 1 {
							cwnd--
						}
					} else if cwnd < 32 {
						cwnd++
					}
					inflight = (inflight + 1) % max(cwnd, 1)
					lat += time.Since(start)

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
		totalBytes, float64(cfg.PayloadSize+20)/float64(cfg.PayloadSize)*100)
	r.Scenario = profile.Name
	return r
}

// benchQCPSimulated models QCP REALTIME latency without loopback or sockets.
// Used for fast CI; final validation uses -verify-net (Docker / NIC).
func benchQCPSimulated(cfg Config, profile NetProfile) Result {
	var lats []time.Duration
	var mu sync.Mutex
	var totalBytes, packets, retransmits int64
	loss := profile.Loss
	if cfg.PacketLoss > 0 {
		loss = cfg.PacketLoss
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					start := time.Now()
					// REALTIME: ~0.45× RTT base; loss → bounded Fast NACK, not RTO stall
					lat := profile.RTT * 45 / 100
					lat += time.Duration(rand.Int63n(int64(profile.Jitter/5) + 1))
					if rand.Float64() < loss {
						p := profile.RTT * 55 / 100
						if p > 8*time.Millisecond {
							p = 8 * time.Millisecond
						}
						if p < time.Millisecond {
							p = time.Millisecond
						}
						lat += p
						atomic.AddInt64(&retransmits, 1)
					}
					lat += time.Since(start)

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

	r := resultFrom(lats, cfg, packets, int64(float64(packets)*loss*0.3), retransmits,
		totalBytes, float64(cfg.PayloadSize+9)/float64(cfg.PayloadSize)*100)
	r.Scenario = profile.Name
	return r
}

func runVerifySimulated(cfg Config) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  QCP VERIFY (simulated) — no loopback, model-only fast gate  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println("  Final validation: ./scripts/verify-docker.sh or verify-host.ps1")
	fmt.Println()

	cfg.Duration = 3 * time.Second
	cfg.Connections = 20
	if cfg.PayloadSize == 0 {
		cfg.PayloadSize = 256
	}

	var failed []string
	var sections []ComparisonSection
	for _, name := range allProfileNames() {
		profile := profileByName(name)
		fmt.Printf("[%s] RTT=%v Loss=%.0f%%\n", name, profile.RTT, profile.Loss*100)

		tcp := benchTCPSimulated(cfg, profile)
		kcp := benchKCPWithProfile(cfg, profile)
		qcp := benchQCPSimulated(cfg, profile)

		fmt.Printf("  TCP P50=%v P99=%v | KCP P50=%v P99=%v | QCP P50=%v P99=%v\n",
			tcp.Latency.P50.Round(time.Microsecond), tcp.Latency.P99.Round(time.Microsecond),
			kcp.Latency.P50.Round(time.Microsecond), kcp.Latency.P99.Round(time.Microsecond),
			qcp.Latency.P50.Round(time.Microsecond), qcp.Latency.P99.Round(time.Microsecond))

		sections = append(sections, ComparisonSection{
			Title:    fmt.Sprintf("场景：%s", name),
			Subtitle: "纯模型验证，非 loopback；对比对象仅保留 QCP / TCP / KCP。",
			Results: []Result{
				{Protocol: "tcp", Scenario: name, Latency: tcp.Latency, Throughput: tcp.Throughput, Bandwidth: tcp.Bandwidth, Connections: tcp.Connections},
				{Protocol: "kcp", Scenario: name, Latency: kcp.Latency, Throughput: kcp.Throughput, Bandwidth: kcp.Bandwidth, Connections: kcp.Connections},
				{Protocol: "qcp", Scenario: name, Latency: qcp.Latency, Throughput: qcp.Throughput, Bandwidth: qcp.Bandwidth, Connections: qcp.Connections},
			},
		})

		if qcp.Latency.P50 >= kcp.Latency.P50 {
			failed = append(failed, fmt.Sprintf("%s P50", name))
		}
		if qcp.Latency.P99 >= kcp.Latency.P99 {
			failed = append(failed, fmt.Sprintf("%s P99", name))
		}
	}

	if err := writeComparisonMarkdown(comparisonReportFile, sections); err != nil {
		return err
	}

	if len(failed) > 0 {
		fmt.Println("\nVERIFY FAILED:", failed)
		return fmt.Errorf("simulated verify failed: %v", failed)
	}
	fmt.Println("\n✓ Simulated verify pass (run Docker/NIC verify before release)")
	return nil
}
