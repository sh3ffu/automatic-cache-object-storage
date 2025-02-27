package objectStorage

import (
	"automatic-cache-object-storage/cache"
	"bytes"
	"io"
	"net/http"
	"strings"
)

// MinioObjectStorageAdapter is an implementation of the ObjectStorageAdapter interface

type MinioObjectStorageAdapter struct {
	Host string
}

func (mosa *MinioObjectStorageAdapter) ExtractObjectKey(req *http.Request) string {
	return req.URL.Host + req.URL.Path + "?" + req.URL.RawQuery
}

func (mosa *MinioObjectStorageAdapter) CreateLocalResponse(object *cache.Object) (*http.Response, error) {
	response := &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(*object.Data)),
		Header:        object.OriginalHeaders,
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

	if req.Method != http.MethodGet {
		return false
	}
	path := strings.Split(req.URL.Path, "/")

	hostOk := strings.Contains(req.Host, mosa.Host)
	requestOk := req.Method == http.MethodGet && len(path) >= 3 && validateBucketName(path[1]) && minIoisObjectKey(getObjectPathFromURL(req.URL.String()))

	return hostOk && requestOk
}

func getObjectPathFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.Join(parts[2:], "/")
}

func NewMinIOAdapter(host string) MinioObjectStorageAdapter {
	return MinioObjectStorageAdapter{
		Host: host,
	}
}
