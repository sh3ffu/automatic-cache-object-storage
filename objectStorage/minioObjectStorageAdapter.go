package objectStorage

import (
	"automatic-cache-object-storage/cache"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MinioObjectStorageAdapter is an implementation of the ObjectStorageAdapter interface

type MinioObjectStorageAdapter struct {
	Host string
}

func (mosa *MinioObjectStorageAdapter) ExtractObjectMeta(req *http.Request) (*cache.ObjectMetadata, error) {
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
		Host:            host,
		Bucket:          bucket,
		Key:             key,
		OriginalHeaders: req.Header,
	}, nil
}

func (mosa *MinioObjectStorageAdapter) CreateLocalResponse(object *cache.Object) (*http.Response, error) {
	response := &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(*object.Data)),
		Header:        object.Metadata.OriginalHeaders,
		Body:          io.NopCloser(bytes.NewReader(*object.Data)),
	}
	return response, nil
}

func minIoisObjectKey(key string) bool {
	return key != ""
}

func (mosa *MinioObjectStorageAdapter) ShouldIntercept(req *http.Request) bool {
	if req.URL.RawQuery == "location" {
		return false
	}
	path := strings.Split(req.URL.Path, "/")

	hostOk := strings.Contains(req.Host, mosa.Host)
	requestOk := req.Method == http.MethodGet && len(path) >= 3 && validateBucketName(path[1]) && minIoisObjectKey(path[2])

	return hostOk && requestOk
}

func NewMinIOAdapter(host string) MinioObjectStorageAdapter {
	return MinioObjectStorageAdapter{
		Host: host,
	}
}
