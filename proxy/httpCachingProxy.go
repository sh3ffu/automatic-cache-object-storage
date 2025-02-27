package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"automatic-cache-object-storage/cache"
	"automatic-cache-object-storage/objectStorage"
)

type HttpCachingProxy struct {
	Cache                 cache.Cache
	ObjectStorageAdapters []objectStorage.ObjectStorage
}

func (p *HttpCachingProxy) handleHttpInternal(conn net.Conn, targetAddr net.Addr) {

	defer conn.Close()

	request, err := http.ReadRequest(bufio.NewReader(conn))

	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	shouldIntercept, adapterIndex := p.shouldIntercept(request)

	if shouldIntercept {
		adapter := p.ObjectStorageAdapters[adapterIndex]
		objKey := adapter.ExtractObjectKey(request)

		initializer := func() (*cache.Object, error) {
			return p.retrieveObjectFromRemote(request, targetAddr, objKey)
		}

		cachedObj, err := p.Cache.Get(objKey, initializer)
		if err == nil {
			// Cache hit - Serve from cache

			response, err := adapter.CreateLocalResponse(cachedObj)
			if err != nil {
				// Log and forward
				log.Printf("Failed to create local response, forwarding connection: %v", err)
				p.forward(conn, targetAddr, request)
				return
			}

			err = response.Write(conn)

			if err != nil {
				log.Printf("Failed to send local response, forwarding connection: %v", err)
				p.forward(conn, targetAddr, request)
			}

			return
		} else {
			// Log and forward
			log.Printf("Failed to retrieve object from cache, forwarding connection: %v", err)
			p.forward(conn, targetAddr, request)
			return
		}

	}

	// Request should not be intercepted - Forward request

	p.forward(conn, targetAddr, request)
}

func (p *HttpCachingProxy) HandleHttp(conn net.Conn, targetAddr net.Addr) {
	p.handleHttpInternal(conn, targetAddr)
}

func (p *HttpCachingProxy) shouldIntercept(req *http.Request) (bool, int) {

	for i, adapter := range p.ObjectStorageAdapters {
		if adapter.ShouldIntercept(req) {
			return true, i
		}
	}
	return false, -1
}
func (p *HttpCachingProxy) retrieveObjectFromRemote(req *http.Request, targetAddr net.Addr, objectKey string) (*cache.Object, error) {

	// Retrieve object from remote storage

	errChan := make(chan error)
	objChan := make(chan *cache.Object)

	go func() {

		targetConn, err := net.Dial("tcp", targetAddr.String())
		if err != nil {
			errChan <- fmt.Errorf("failed to connect to target: %v", err)
			return
		}
		defer targetConn.Close()

		err = req.Write(targetConn)
		if err != nil {
			errChan <- fmt.Errorf("failed to send request to target: %v", err)
			return
		}

		res, err := http.ReadResponse(bufio.NewReader(targetConn), req)
		if err != nil {
			errChan <- fmt.Errorf("failed to read response from target: %v", err)
			return
		}

		if res.StatusCode != http.StatusOK {
			body, err := io.ReadAll(res.Body)
			if err != nil {
				errChan <- fmt.Errorf("could not read non-OK response body")
				return
			}
			errChan <- fmt.Errorf("received non-OK HTTP status: %s : %s", res.Status, string(body))
			return
		}

		var buffer bytes.Buffer
		_, err = buffer.ReadFrom(res.Body)

		if err != nil {
			errChan <- err
			return
		}
		data := buffer.Bytes()
		object := &cache.Object{
			Key:             objectKey,
			OriginalHeaders: res.Header,
			Data:            &data,
		}
		errChan <- nil
		objChan <- object
	}()

	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("failed to retrieve object from remote: %v", err)
	}
	return <-objChan, nil
}

func (p *HttpCachingProxy) forward(conn net.Conn, targetAddr net.Addr, req *http.Request) {

	targetConn, err := net.Dial("tcp", targetAddr.String())
	if err != nil {
		conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	// Forward request
	req.Write(targetConn)

	// Forward response
	res, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		return
	}
	res.Write(conn)
}

func NewHttpCachingProxy(cache cache.Cache, objectStorageAdapters []objectStorage.ObjectStorage) *HttpCachingProxy {
	return &HttpCachingProxy{
		Cache:                 cache,
		ObjectStorageAdapters: objectStorageAdapters,
	}
}
