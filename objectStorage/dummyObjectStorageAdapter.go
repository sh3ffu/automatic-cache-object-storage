package objectStorage

import (
	"automatic-cache-object-storage/cache"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type DummyObjectStorageAdapter struct {
	Host string
}

func (dosa *DummyObjectStorageAdapter) ExtractObjectMeta(req *http.Request) (*cache.ObjectMetadata, error) {
	host := req.Host
	path := strings.Split(req.URL.Path, "/")

	if len(path) < 3 {
		return nil, fmt.Errorf("invalid request: %s", req.URL.Path)
	}

	if req.Method != http.MethodGet {
		return nil, fmt.Errorf("invalid request method. Only GET requests accepted: %s", req.Method)
	}

	bucket := path[1]
	key := path[2]

	if host == "" || bucket == "" || key == "" {
		return nil, fmt.Errorf("invalid request: some fields are empty")
	}

	return &cache.ObjectMetadata{
		Host:   host,
		Bucket: bucket,
		Key:    key,
	}, nil
}

func (dosa *DummyObjectStorageAdapter) CreateLocalResponse(object *cache.Object) (*http.Response, error) {

	response := &http.Response{
		Header:        http.Header{},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(*object.Data)),
		Body:          io.NopCloser(bytes.NewReader(*object.Data)),
	}

	response.Header.Add("Content-Type", "application/octet-stream")

	//print response

	// var stringBuilder strings.Builder

	// response.Write(&stringBuilder)

	// log.Default().Println("response: " + stringBuilder.String())

	return response, nil
}

func validateBucketName(bucket string) bool {
	return bucket != "" && bucket != ".." && bucket != "."
}

func validateObjectKey(key string) bool {
	return key != "" && key != ".." && key != "."
}

func (dosa *DummyObjectStorageAdapter) ShouldIntercept(req *http.Request) bool {

	path := strings.Split(req.URL.Path, "/")

	hostOk := strings.Contains(req.Host, dosa.Host)
	requestOk := req.Method == http.MethodGet && len(path) >= 3 && validateBucketName(path[1]) && validateObjectKey(path[2])

	return hostOk && requestOk
}

func NewDummyObjectStorageAdapter(host string) DummyObjectStorageAdapter {
	return DummyObjectStorageAdapter{
		Host: host,
	}
}
