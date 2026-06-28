// KCP baseline — TEST OPPONENT ONLY.
//
// This file simulates KCP-style ARQ behavior for benchmark comparison.
// It is NOT part of QCP. Production games use QCP (qcp-lib-*), not KCP.
package main

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// KCPPacket mimics the 24-byte KCP header for latency simulation.
type KCPPacket struct {
	Conv    uint32
	Cmd     byte
	Frg     byte
	Wnd     uint16
	TS      uint32
	SN      uint32
	una     uint32
	Len     uint32
	Payload []byte
}

func (p *KCPPacket) Marshal() []byte {
	buf := make([]byte, 24+len(p.Payload))
	binary.LittleEndian.PutUint32(buf[0:4], p.Conv)
	buf[4] = p.Cmd
	buf[5] = p.Frg
	binary.LittleEndian.PutUint16(buf[6:8], p.Wnd)
	binary.LittleEndian.PutUint32(buf[8:12], p.TS)
	binary.LittleEndian.PutUint32(buf[12:16], p.SN)
	binary.LittleEndian.PutUint32(buf[16:20], p.una)
	binary.LittleEndian.PutUint32(buf[20:24], p.Len)
	copy(buf[24:], p.Payload)
	return buf
}

func benchKCP(cfg Config) Result {
	return benchKCPWithProfile(cfg, profileByName("lan"))
}

func benchKCPWithProfile(cfg Config, profile NetProfile) Result {
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
			cwnd, sent := 32, 0
			for {
				select {
				case <-done:
					return
				default:
					start := time.Now()

					// KCP baseline: synthetic RTT + header + RTO on loss
					lat := profile.RTT + time.Duration(100+rand.Intn(200))*time.Microsecond
					_ = payload

					if rand.Float64() < loss {
						lat += kcpLossPenalty(profile)
						atomic.AddInt64(&retransmits, 1)
					}

					sent++
					if sent > cwnd {
						lat += time.Duration(1+rand.Intn(3)) * time.Millisecond
						cwnd = max(cwnd-1, 1)
					} else {
						cwnd = min(cwnd+1, 64)
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

	r := resultFrom(lats, cfg, packets, int64(float64(packets)*loss), retransmits,
		totalBytes, float64(cfg.PayloadSize+24)/float64(cfg.PayloadSize)*100)
	r.Scenario = profile.Name
	return r
}
