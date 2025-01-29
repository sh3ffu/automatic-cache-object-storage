package proxy

import "net"

type HttpProxy interface {
	HandleHttp(conn net.Conn, targetAddr net.Addr)
}
