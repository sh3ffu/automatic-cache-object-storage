package proxy

import "net"

type HttpProxy interface {
	HandleHttp(conn net.Conn, targetAddr net.Addr)
}

type HttpTimedProxy interface {
	HandleHttp(conn net.Conn, targetAddr net.Addr) ProxyStatsEntry
}

type NetworkTimer struct {
	ReadRequest   int64 //Time to read a request from incoming connection
	DialRemote    int64 //Time needed to initiate a TCP connection between proxy and target server (remote object storage)
	WriteRequest  int64 //Time to write the request to outgoing connection
	ReadResponse  int64 //Time to read the response from outgoing connection
	WriteResponse int64 //Time to write the response back to the incoming connection
}

type ProxyStatsEntry struct {
	NetworkTimer

	Total int64 //Total time

	CacheRetrieve int64 //Time to retrieve an object from cache (excluding initializing)
	Initialize    int64 //Time to initialize an object in cache (after a cache miss)

	CacheMiss bool
	Forwarded bool
	Failed    bool

	ObjectKey string
	WorkerID  int
}
