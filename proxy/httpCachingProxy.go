package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"automatic-cache-object-storage/cache"
	"automatic-cache-object-storage/objectStorage"
)

type HttpCachingProxy struct {
	Cache                 cache.Cache
	ObjectStorageAdapters []objectStorage.ObjectStorage
}

func calculateKey(meta *cache.ObjectMetadata) string {
	return meta.Host + "/" + meta.Bucket + "/" + meta.Key
}

//var connectionCounter uint64 = 0

func (p *HttpCachingProxy) handleHttpInternal(conn net.Conn, targetAddr net.Addr) {

	defer conn.Close()

	//TODO: try to use raw request instead of http.Request
	request, _, err := pipeRequest(conn)

	if err != nil {
		//TODO: forward request directly
		return
	}

	fixRequest(request)

	shouldIntercept, adapterIndex := p.shouldIntercept(request)

	if shouldIntercept {
		adapter := p.ObjectStorageAdapters[adapterIndex]
		objectMeta, err := adapter.ExtractObjectMeta(request)

		if err != nil {
			p.forward(conn, targetAddr, request)
			return
		}

		initializer := func() (*cache.Object, error) {
			return p.retrieveObjectFromRemote(request, *objectMeta)
		}

		cachedObj, err := p.Cache.Get(calculateKey(objectMeta), initializer)
		if err == nil {
			// Cache hit - Serve from cache
			response, err := adapter.CreateLocalResponse(cachedObj)
			if err != nil {
				// Log and forward
				log.Printf("Failed to create local response, forwarding connection: %v", err)
				p.forward(conn, targetAddr, request)
				return
			}

			errChan := make(chan error)
			var wg sync.WaitGroup

			go p.sendLocalResponse(conn, response, errChan, &wg)

			wg.Wait()
			if err := <-errChan; err != nil {
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
	// Increment the counter for each http request
	//connectionCounter++
	//startTime := time.Now()
	p.handleHttpInternal(conn, targetAddr)
	//log.Printf("Request %d took %v", connectionCounter, time.Since(startTime))
}

func (p *HttpCachingProxy) shouldIntercept(req *http.Request) (bool, int) {
	for i, adapter := range p.ObjectStorageAdapters {
		if adapter.ShouldIntercept(req) {
			return true, i
		}
	}
	return false, -1
}

func (p *HttpCachingProxy) sendLocalResponse(conn net.Conn, res *http.Response, errChan chan error, wg *sync.WaitGroup) {
	wg.Add(1)
	err := res.Write(conn)
	errChan <- err
}

func (p *HttpCachingProxy) retrieveObjectFromRemote(req *http.Request, objectMeta cache.ObjectMetadata) (*cache.Object, error) {

	// Retrieve object from remote storage

	errChan := make(chan error)
	objChan := make(chan *cache.Object)

	go func() {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer res.Body.Close()

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
			Metadata: &objectMeta,
			Data:     &data,
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

	// Pipe data between connections
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, targetConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(targetConn, conn)
	}()

	wg.Wait()
}

func NewHttpCachingProxy(cache cache.Cache, objectStorageAdapters []objectStorage.ObjectStorage) *HttpCachingProxy {
	return &HttpCachingProxy{
		Cache:                 cache,
		ObjectStorageAdapters: objectStorageAdapters,
	}
}

func pipeRequest(conn net.Conn) (*http.Request, *bytes.Buffer, error) {

	var buffer bytes.Buffer

	parserTee := io.TeeReader(conn, &buffer)

	request, err := http.ReadRequest(bufio.NewReader(parserTee))

	if err != nil {
		return nil, nil, err
	}

	return request, &buffer, nil

}
