package cache

import (
	"bytes"
	"errors"
	"io"
	"log"
	"testing"
	"time"
)

func TestDummyPrinterCache_Get(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	data := []byte("testData")
	obj := &Object{
		Key:  "localhost/testBucket/testKey",
		Data: &data,
	}

	cache.lock.Lock()
	cache.store[obj.Key] = obj
	cache.lock.Unlock()

	initializer := func() (*Object, error) {
		return obj, nil
	}

	retrievedObj, err := cache.Get(obj.Key, initializer)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if retrievedObj != obj {
		t.Errorf("Expected object %v, but got %v", obj, retrievedObj)
	}

	bytes, err := io.ReadAll(bytes.NewReader(*retrievedObj.Data))
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if string(bytes) != "testData" {
		t.Errorf("Expected object data 'testData', but got %s", string(bytes))
	}
}

func TestDummyPrinterCache_Get_NotFound(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	initializer := func() (*Object, error) {
		time.Sleep(500 * time.Millisecond)
		return nil, errors.New("Retrieval failed")
	}

	retrievedObj, err := cache.Get("dummy_key", initializer)
	if err == nil {
		t.Errorf("Expected an error, but got nil")
	}
	if retrievedObj != nil {
		t.Errorf("Expected nil object, but got %v", retrievedObj)
	}
}

func TestDummyPrinterCache_Get_Initializer(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	data := []byte("testData")
	object := &Object{
		Key:  "localhost/testBucket/testKey",
		Data: &data,
	}

	initializer := func() (*Object, error) {
		time.Sleep(500 * time.Millisecond)
		return object, nil
	}

	retrievedObj, err := cache.Get(object.Key, initializer)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if retrievedObj != object {
		t.Errorf("Expected object %v, but got %v", object, retrievedObj)
	}

	bytes, err := io.ReadAll(bytes.NewReader(*retrievedObj.Data))
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if string(bytes) != "testData" {
		t.Errorf("Expected object data 'testData', but got %s", string(bytes))
	}
}

func TestDummyPrinterCache_Put(t *testing.T) {
	cache := &DummyPrinterCache{
		logger:  log.New(io.Discard, "", log.LstdFlags),
		maxSize: 1024,
		store:   make(map[string]*Object),
	}

	data := []byte("testData")
	obj := &Object{
		Key:  "localhost/testBucket/testKey",
		Data: &data,
	}

	err := cache.Put(obj)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	cache.lock.RLock()
	storedObj, exists := cache.store[obj.Key]
	cache.lock.RUnlock()

	if !exists {
		t.Errorf("Expected object to be stored in the cache")
	}
	if storedObj != obj {
		t.Errorf("Expected stored object %v, but got %v", obj, storedObj)
	}
	if storedObj.Key != obj.Key {
		t.Errorf("Expected stored object key %s, but got %s", obj.Key, storedObj.Key)
	}
	if !bytes.Equal(*storedObj.Data, data) {
		t.Errorf("Expected stored object data %v, but got %v", data, *storedObj.Data)
	}
}
