package main

import (
	"math/rand"
	"net"
	"sync"
	"time"
)

// NetProfile describes impaired network conditions.
type NetProfile struct {
	Name    string
	RTT     time.Duration
	Jitter  time.Duration
	Loss    float64
	Reorder float64
}

var netProfiles = map[string]NetProfile{
	"lan":       {Name: "lan", RTT: 2 * time.Millisecond, Jitter: 500 * time.Microsecond, Loss: 0},
	"wifi":      {Name: "wifi", RTT: 20 * time.Millisecond, Jitter: 5 * time.Millisecond, Loss: 0.01},
	"4g":        {Name: "4g", RTT: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Loss: 0.02},
	"3g":        {Name: "3g", RTT: 150 * time.Millisecond, Jitter: 30 * time.Millisecond, Loss: 0.05},
	"congested": {Name: "congested", RTT: 100 * time.Millisecond, Jitter: 40 * time.Millisecond, Loss: 0.10},
	"extreme":   {Name: "extreme", RTT: 300 * time.Millisecond, Jitter: 80 * time.Millisecond, Loss: 0.20},
}

func profileByName(name string) NetProfile {
	if p, ok := netProfiles[name]; ok {
		return p
	}
	return NetProfile{Name: "custom", RTT: 0, Loss: 0}
}

func allProfileNames() []string {
	return []string{"lan", "wifi", "4g", "3g", "congested", "extreme"}
}

type udpImpairProxy struct {
	listen   *net.UDPConn
	target   *net.UDPAddr
	profile  NetProfile
	upstream map[string]*net.UDPConn
	mu       sync.Mutex
	stop     chan struct{}
}

func startImpairProxy(listenAddr, targetAddr string, profile NetProfile) (*udpImpairProxy, error) {
	laddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	taddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}
	p := &udpImpairProxy{
		listen:   conn,
		target:   taddr,
		profile:  profile,
		upstream: make(map[string]*net.UDPConn),
		stop:     make(chan struct{}),
	}
	go p.run()
	return p, nil
}

func (p *udpImpairProxy) Close() error {
	close(p.stop)
	return p.listen.Close()
}

func (p *udpImpairProxy) run() {
	buf := make([]byte, 2048)
	for {
		n, clientAddr, err := p.listen.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-p.stop:
				return
			default:
				continue
			}
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		go p.forwardToTarget(data, clientAddr)
	}
}

func (p *udpImpairProxy) getUpstream(clientAddr *net.UDPAddr) (*net.UDPConn, error) {
	key := clientAddr.String()
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.upstream[key]; ok {
		return c, nil
	}
	c, err := net.DialUDP("udp", nil, p.target)
	if err != nil {
		return nil, err
	}
	p.upstream[key] = c
	go p.readUpstream(c, clientAddr)
	return c, nil
}

func (p *udpImpairProxy) readUpstream(up *net.UDPConn, clientAddr *net.UDPAddr) {
	buf := make([]byte, 2048)
	for {
		up.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := up.Read(buf)
		if err != nil {
			select {
			case <-p.stop:
				return
			default:
				continue
			}
		}
		p.forwardToClient(buf[:n], clientAddr)
	}
}

func (p *udpImpairProxy) forwardToTarget(data []byte, clientAddr *net.UDPAddr) {
	if p.shouldDrop() {
		return
	}
	time.Sleep(p.oneWayDelay())
	up, err := p.getUpstream(clientAddr)
	if err != nil {
		return
	}
	up.Write(data)
}

func (p *udpImpairProxy) forwardToClient(data []byte, clientAddr *net.UDPAddr) {
	if p.shouldDrop() {
		return
	}
	delay := p.oneWayDelay()
	if p.profile.Reorder > 0 && rand.Float64() < p.profile.Reorder {
		delay += p.profile.Jitter * 2
	}
	time.Sleep(delay)
	p.listen.WriteToUDP(data, clientAddr)
}

func (p *udpImpairProxy) shouldDrop() bool {
	return p.profile.Loss > 0 && rand.Float64() < p.profile.Loss
}

func (p *udpImpairProxy) oneWayDelay() time.Duration {
	half := p.profile.RTT / 2
	jitter := time.Duration(0)
	if p.profile.Jitter > 0 {
		jitter = time.Duration(rand.Int63n(int64(p.profile.Jitter)))
	}
	return half + jitter
}

func kcpLossPenalty(profile NetProfile) time.Duration {
	base := 8 * time.Millisecond
	if profile.RTT > 50*time.Millisecond {
		base = 20 * time.Millisecond
	}
	return base + time.Duration(rand.Intn(12))*time.Millisecond
}

func qcpLossPenalty(profile NetProfile) time.Duration {
	penalty := time.Duration(float64(profile.RTT) * 1.1)
	max := 8 * time.Millisecond
	if profile.RTT > max {
		max = profile.RTT
	}
	if penalty > max {
		penalty = max
	}
	if penalty < time.Millisecond {
		penalty = time.Millisecond
	}
	return penalty
}
