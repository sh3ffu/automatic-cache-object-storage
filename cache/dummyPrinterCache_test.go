package cache

import (
	"bytes"
	"io"
	"log"
	"testing"
)

func TestDummyPrinterCache_GetMetadata(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	meta := &ObjectMetadata{
		host:   "localhost",
		bucket: "testBucket",
		key:    "testKey",
	}
	obj := &Object{
		Metadata: meta,
		Reader:   bytes.NewReader([]byte("testData")),
	}

	cache.lock.Lock()
	cache.store[calculateKey(meta)] = obj
	cache.lock.Unlock()

	retrievedMeta, err := cache.GetMetadata(calculateKey(meta))
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if retrievedMeta != meta {
		t.Errorf("Expected metadata %v, but got %v", meta, retrievedMeta)
	}
}

func TestDummyPrinterCache_Get(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	meta := &ObjectMetadata{
		host:   "localhost",
		bucket: "testBucket",
		key:    "testKey",
	}
	obj := &Object{
		Metadata: meta,
		Reader:   bytes.NewReader([]byte("testData")),
	}

	cache.lock.Lock()
	cache.store[calculateKey(meta)] = obj
	cache.lock.Unlock()

	retrievedObj, err := cache.Get(calculateKey(meta))
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if retrievedObj != obj {
		t.Errorf("Expected object %v, but got %v", obj, retrievedObj)
	}
}

func TestDummyPrinterCache_Put(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	meta := &ObjectMetadata{
		host:   "localhost",
		bucket: "testBucket",
		key:    "testKey",
	}
	obj := &Object{
		Metadata: meta,
		Reader:   bytes.NewReader([]byte("testData")),
	}

	err := cache.Put(obj)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	cache.lock.RLock()
	storedObj, exists := cache.store[calculateKey(meta)]
	cache.lock.RUnlock()

	if !exists {
		t.Errorf("Expected object to be stored in the cache")
	}
	if storedObj != obj {
		t.Errorf("Expected stored object %v, but got %v", obj, storedObj)
	}
}
