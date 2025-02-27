package objectStorage

import (
	"automatic-cache-object-storage/cache"
	"net/http"
)

type ObjectStorage interface {
	ShouldIntercept(req *http.Request) bool
	ExtractObjectKey(req *http.Request) string
	CreateLocalResponse(object *cache.Object) (*http.Response, error)
}
