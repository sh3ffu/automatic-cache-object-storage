package cache

import (
	"io"
)

type ObjectMetadata struct {
	host   string
	bucket string
	key    string
}

type Object struct {
	io.Reader
	io.Writer
	Metadata *ObjectMetadata
}

type Cache interface {
	Get(key string) (*Object, error)
	GetMetadata(key string) (*ObjectMetadata, error)
	Put(*Object) error
}
