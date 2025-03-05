package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"automatic-cache-object-storage/cache"
	"automatic-cache-object-storage/objectStorage"
)

type HttpCachingTimedProxy struct {
	Cache                 cache.Cache
	ObjectStorageAdapters []objectStorage.ObjectStorage
}

func (p *HttpCachingTimedProxy) handleHttpTimed(conn net.Conn, targetAddr net.Addr) ProxyStatsEntry {

	defer conn.Close()

	stats := ProxyStatsEntry{}

	// Read request
	startTime := time.Now()
	t := startTime
	request, err := http.ReadRequest(bufio.NewReader(conn))
	stats.ReadRequest = time.Since(t).Nanoseconds()

	if err != nil {
		t = time.Now()
		if err == io.ErrUnexpectedEOF {
			log.Printf("Unexpected EOF while reading request")
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return stats
		}
		log.Printf("Failed to read request: %v", err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		stats.WriteResponse = time.Since(t).Nanoseconds()
		return stats
	}

	shouldIntercept, adapterIndex := p.shouldIntercept(request)

	if shouldIntercept {
		adapter := p.ObjectStorageAdapters[adapterIndex]
		objKey := adapter.ExtractObjectKey(request)

		stats.ObjectKey = objKey

		initializer := func() (*cache.Object, error) {
			return p.retrieveObjectFromRemoteTimed(request, targetAddr, objKey, &stats.NetworkTimer)
		}

		cachedObj, cacheRetrieveT, initT, err := p.Cache.GetTimed(objKey, initializer)
		stats.CacheRetrieve = cacheRetrieveT
		stats.Initialize = initT
		stats.CacheMiss = initT > 0

		if err == nil {
			// Cache hit - Serve from cache

			response, err := adapter.CreateLocalResponse(cachedObj)
			if err != nil {
				// Log and forward
				log.Printf("Failed to create local response, forwarding connection: %v", err)
				stats.Forwarded = true
				stats.Failed = true
				p.forwardTimed(conn, targetAddr, request, &stats.NetworkTimer)
				return stats
			}

			t = time.Now()
			err = response.Write(conn)
			stats.WriteResponse = time.Since(t).Nanoseconds()

			if err != nil {
				log.Printf("Failed to send local response, forwarding connection: %v", err)
				stats.Forwarded = true
				stats.Failed = true
				p.forwardTimed(conn, targetAddr, request, &stats.NetworkTimer)
			}

			return stats
		} else {

			//DEPRECATED - REMOVE
			// Log and forward
			log.Printf("Failed to retrieve object from cache, forwarding connection: %v", err)
			stats.Forwarded = true
			p.forwardTimed(conn, targetAddr, request, &stats.NetworkTimer)
			return stats
		}

	}

	// Request should not be intercepted - Forward request

	p.forwardTimed(conn, targetAddr, request, &stats.NetworkTimer)
	return stats
}

func (p *HttpCachingTimedProxy) HandleHttp(conn net.Conn, targetAddr net.Addr) ProxyStatsEntry {
	stats := p.handleHttpTimed(conn, targetAddr)
	return stats
}

func (p *HttpCachingTimedProxy) shouldIntercept(req *http.Request) (bool, int) {

	for i, adapter := range p.ObjectStorageAdapters {
		if adapter.ShouldIntercept(req) {
			return true, i
		}
	}
	return false, -1
}

func (p *HttpCachingTimedProxy) retrieveObjectFromRemoteTimed(req *http.Request, targetAddr net.Addr, objectKey string, times *NetworkTimer) (*cache.Object, error) {

	// Retrieve object from remote storage

	//TODO: Think about changing to synchronous

	localTimes := NetworkTimer{}

	start := time.Now()
	targetConn, err := net.Dial("tcp", targetAddr.String())
	localTimes.DialRemote = time.Since(start).Nanoseconds()

	if err != nil {
		return nil, fmt.Errorf("failed to connect to target: %v", err)
	}
	defer targetConn.Close()

	start = time.Now()
	err = req.Write(targetConn)
	localTimes.WriteRequest = time.Since(start).Nanoseconds()
	if err != nil {
		return nil, fmt.Errorf("failed to send request to target: %v", err)
	}

	start = time.Now()
	res, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from target: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read non-OK response body")
		}
		return nil, fmt.Errorf("received non-OK HTTP status: %s : %s", res.Status, string(body))
	}

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(res.Body)
	localTimes.ReadResponse = time.Since(start).Nanoseconds()

	if err != nil {
		return nil, err
	}
	data := buffer.Bytes()
	object := &cache.Object{
		Key:             objectKey,
		OriginalHeaders: res.Header,
		Data:            &data,
	}
	// errChan <- nil
	// objChan <- object
	// timerChan <- localTimes

	times.DialRemote = localTimes.DialRemote
	times.WriteRequest = localTimes.WriteRequest
	times.ReadRequest = localTimes.ReadRequest
	times.ReadResponse = localTimes.ReadResponse

	return object, nil
}

func (p *HttpCachingTimedProxy) forwardTimed(conn net.Conn, targetAddr net.Addr, req *http.Request, times *NetworkTimer) {

	//Dial remote
	start := time.Now()
	targetConn, err := net.Dial("tcp", targetAddr.String())
	times.DialRemote = time.Since(start).Nanoseconds()

	if err != nil {
		start = time.Now()
		conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		times.WriteResponse = time.Since(start).Nanoseconds()
		return
	}
	defer targetConn.Close()

	// Forward request
	start = time.Now()
	req.Write(targetConn)
	times.WriteRequest = time.Since(start).Nanoseconds()

	// Forward response
	start = time.Now()
	res, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	times.ReadResponse = time.Since(start).Nanoseconds()
	if err != nil {
		start = time.Now()
		conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		times.WriteResponse = time.Since(start).Nanoseconds()
		return
	}
	start = time.Now()
	res.Write(conn)
	times.WriteResponse = time.Since(start).Nanoseconds()
}

func NewHttpCachingTimedProxy(cache cache.Cache, objectStorageAdapters []objectStorage.ObjectStorage) *HttpCachingTimedProxy {
	return &HttpCachingTimedProxy{
		Cache:                 cache,
		ObjectStorageAdapters: objectStorageAdapters,
	}
}
