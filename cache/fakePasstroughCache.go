package cache

import (
	"fmt"
	"log"
)

// FakePasstroughCache is a dummy implementation of the Cache interface.
// It is not caching any object, only mocking the calls to a cache.

type FakePasstroughCache struct {
	logger  *log.Logger
	maxSize int64
}

func (pc *FakePasstroughCache) GetMetadata(key string) (*ObjectMetadata, error) {
	return nil, fmt.Errorf("Object with key %s not found", key)
}

func (pc *FakePasstroughCache) Get(key string, initializer Initializer) (*Object, error) {
	obj, exists := pc.get(key)
	if !exists {
		return pc.initialize(key, initializer)
	}

	return obj, nil
}

func (pc *FakePasstroughCache) Put(o *Object) error {
	pc.put(o)

	return nil

}

func (pc *FakePasstroughCache) get(key string) (*Object, bool) {
	return nil, false
}

func (pc *FakePasstroughCache) put(o *Object) {
}

func (pc *FakePasstroughCache) initialize(key string, initializer Initializer) (*Object, error) {

	data, exists := pc.get(key)
	if exists {
		return data, nil
	}

	if initializer != nil {

		obj, err := initializer()
		if err != nil {
			return nil, err
		}
		pc.put(obj)
		return obj, nil

	}
	return nil, fmt.Errorf("Object with key %s not found", key)
}

func NewFakePasstroughCache(logger *log.Logger, maxSize int64) *FakePasstroughCache {
	return &FakePasstroughCache{
		logger:  logger,
		maxSize: maxSize,
	}
}
