package main

import (
	"time"

	"github.com/neko233-com/qcp-lib-go/qcp"
)

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
