package cache

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInitializerNil  = errors.New("initializer is nil")
	ErrInvalidKey      = errors.New("key is invalid")
	ErrObjectNil       = errors.New("object is nil")
	ErrDataNil         = errors.New("data is nil")
	ErrMetadataNil     = errors.New("metadata is nil")
	ErrInitializer     = errors.New("initializer error")
	ErrCacheMiss       = errors.New("cache miss")
	ErrSerialization   = errors.New("serialization error")
	ErrDeserialization = errors.New("deserialization error")
)

type ObjectMetadata struct {
	Host            string
	Bucket          string
	Key             string
	OriginalHeaders map[string][]string
}

type Object struct {
	Data     *[]byte
	Metadata *ObjectMetadata
}

type Initializer func() (*Object, error)

type Cache interface {
	Get(key string, initializer Initializer) (*Object, error)
	GetMetadata(key string) (*ObjectMetadata, error)
	Put(*Object) error
}

func NewMetadata(key string) (*ObjectMetadata, error) {
	splitted := strings.Split(key, "/")
	if len(splitted) != 3 {
		return nil, fmt.Errorf("invalid key format: %s", key)
	}
	return &ObjectMetadata{
		Host:   splitted[0],
		Bucket: splitted[1],
		Key:    splitted[2],
	}, nil

}
