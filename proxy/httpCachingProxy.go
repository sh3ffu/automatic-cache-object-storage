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

type HttpCachingProxy struct {
	Cache                 cache.Cache
	ObjectStorageAdapters []objectStorage.ObjectStorage
}

var failResponse = http.Response{
	StatusCode: http.StatusServiceUnavailable,
	Proto:      "HTTP/1.1",
	ProtoMajor: 1,
	ProtoMinor: 1,
	Header:     make(http.Header),
	Body:       io.NopCloser(bytes.NewReader([]byte("Service Unavailable"))),
}

func (p *HttpCachingProxy) handleHttpInternal(conn net.Conn, targetAddr net.Addr) {

	defer conn.Close()

	request, err := http.ReadRequest(bufio.NewReader(conn))

	if err != nil {
		log.Printf("Failed to read request: %v", err)
		failResponse.Write(conn)
		return
	}

	shouldIntercept, adapterIndex := p.shouldIntercept(request)

	if shouldIntercept {
		adapter := p.ObjectStorageAdapters[adapterIndex]
		objKey := adapter.ExtractObjectKey(request)

		cachedObj, err := p.Cache.Get(objKey)
		if err == nil {
			// Cache hit - Serve from cache
			//log.Printf("Cache hit for object: %s", objKey)

			response, err := adapter.CreateLocalResponse(cachedObj)
			if err != nil {
				// Log and forward
				log.Printf("Failed to create local response, forwarding connection: %v", err)
				p.forward(conn, targetAddr, request)
				return
			}

			t := time.Now()
			err = response.Write(conn)
			end := time.Since(t).Milliseconds()

			logIfSuspicious(end, "intercept", objKey)

			if err != nil {
				log.Printf("Failed to send local response, forwarding connection: %v", err)
				p.forward(conn, targetAddr, request)
			}

			return
		} else if err == cache.ErrCacheMiss {
			// Cache miss - Retrieve object from remote storage
			//log.Printf("Cache miss for object: %s", objKey)
			cachingErr := p.retrieveAndCacheObjectFromRemote(request, conn, targetAddr, objKey)
			if cachingErr != nil {
				log.Printf("Failed to cache object: %v", cachingErr)
				p.forward(conn, targetAddr, request)
			}
			return
		} else {
			log.Printf("Failed to retrieve object from cache, forwarding connection: %v", err)
		}

	}

	// Request should not be intercepted or unknown error occured - Forward request

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

func (p *HttpCachingProxy) retrieveAndCacheObjectFromRemote(req *http.Request, conn net.Conn, targetAddr net.Addr, objectKey string) error {

	t := time.Now()
	targetConn, err := net.Dial("tcp", targetAddr.String())
	logIfSuspicious(time.Since(t).Milliseconds(), "retrieveAndCache-dial", objectKey)
	if err != nil {
		return fmt.Errorf("failed to connect to target: %v", err)
	}
	defer targetConn.Close()

	t = time.Now()
	err = req.Write(targetConn)
	logIfSuspicious(time.Since(t).Milliseconds(), "retrieveAndCache-writeReq", objectKey)
	if err != nil {
		return fmt.Errorf("failed to send request to target: %v", err)
	}

	tee := io.TeeReader(targetConn, conn)

	t = time.Now()
	res, err := http.ReadResponse(bufio.NewReader(tee), req)
	logIfSuspicious(time.Since(t).Milliseconds(), "retrieveAndCache-tee-readResp", objectKey)
	if err != nil {
		return fmt.Errorf("failed to read response from target: %v", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("could not read non-OK response body")
		}
		return fmt.Errorf("received non-OK HTTP status: %s : %s", res.Status, string(body))
	}

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(res.Body)

	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	data := buffer.Bytes()

	object := &cache.Object{
		Key:             objectKey,
		OriginalHeaders: res.Header,
		Data:            &data,
	}
	err = p.Cache.Put(object)
	if err != nil {
		return fmt.Errorf("failed to cache object: %v", err)
	}
	return nil
}

func (p *HttpCachingProxy) forward(conn net.Conn, targetAddr net.Addr, req *http.Request) {

	targetConn, err := net.Dial("tcp", targetAddr.String())
	if err != nil {
		failResponse.Write(conn)
		return
	}
	defer targetConn.Close()

	// Forward request
	t := time.Now()
	req.Write(targetConn)
	logIfSuspicious(time.Since(t).Milliseconds(), "forward", req.URL.Path)

	tee := io.TeeReader(targetConn, conn)
	//Forward response
	http.ReadResponse(bufio.NewReader(tee), req)
}

func NewHttpCachingProxy(cache cache.Cache, objectStorageAdapters []objectStorage.ObjectStorage) *HttpCachingProxy {
	return &HttpCachingProxy{
		Cache:                 cache,
		ObjectStorageAdapters: objectStorageAdapters,
	}
}

func logIfSuspicious(time int64, action string, objKey string) {
	if time > 1000 {
		log.Printf("Suspiciously long time to %s object %s: %d ms", action, objKey, time)
	}
}
