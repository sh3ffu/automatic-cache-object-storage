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
	Metadata ObjectMetadata
	Reader   io.Reader
	Writer   io.Writer
}

type Cache interface {
	get(key string) Object
	getMetadata(key string) ObjectMetadata
	put() Object
}
