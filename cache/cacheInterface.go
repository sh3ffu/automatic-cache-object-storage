package cache

import (
	"errors"
)

var (
	ErrInitializerNil  = errors.New("initializer is nil")
	ErrInvalidKey      = errors.New("key is invalid")
	ErrObjectNil       = errors.New("object is nil")
	ErrDataNil         = errors.New("data is nil")
	ErrInitializer     = errors.New("initializer error")
	ErrCacheMiss       = errors.New("cache miss")
	ErrSerialization   = errors.New("serialization error")
	ErrDeserialization = errors.New("deserialization error")
)

type Object struct {
	Key             string
	Data            *[]byte
	OriginalHeaders map[string][]string
}

type Initializer func() (*Object, error)

type Cache interface {
	Get(key string, initializer Initializer) (*Object, error)
	Put(*Object) error
}
