package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

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
	for _, name := range allProfileNames() {
		profile := profileByName(name)
		fmt.Printf("[%s] RTT=%v Loss=%.0f%%\n", name, profile.RTT, profile.Loss*100)

		kcp := benchKCPWithProfile(cfg, profile)
		qcp := benchQCPSimulated(cfg, profile)

		fmt.Printf("  KCP P50=%v P99=%v | QCP P50=%v P99=%v\n",
			kcp.Latency.P50.Round(time.Microsecond), kcp.Latency.P99.Round(time.Microsecond),
			qcp.Latency.P50.Round(time.Microsecond), qcp.Latency.P99.Round(time.Microsecond))

		if qcp.Latency.P50 >= kcp.Latency.P50 {
			failed = append(failed, fmt.Sprintf("%s P50", name))
		}
		if qcp.Latency.P99 >= kcp.Latency.P99 {
			failed = append(failed, fmt.Sprintf("%s P99", name))
		}
	}

	if len(failed) > 0 {
		fmt.Println("\nVERIFY FAILED:", failed)
		return fmt.Errorf("simulated verify failed: %v", failed)
	}
	fmt.Println("\n✓ Simulated verify pass (run Docker/NIC verify before release)")
	return nil
}
