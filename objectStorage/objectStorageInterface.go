package objectStorage

import (
	"automatic-cache-object-storage/cache"
	"net/http"
)

type ObjectStorage interface {
	ShouldIntercept(req *http.Request) bool
	ExtractObjectMeta(req *http.Request) (*cache.ObjectMetadata, error)
	CreateLocalResponse(object *cache.Object) (*http.Response, error)
}
