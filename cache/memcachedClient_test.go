package cache

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/daangn/minimemcached"
)

func TestInitialize(t *testing.T) {

	cfg := &minimemcached.Config{
		Port: 11212,
	}
	mockMemcached, err := minimemcached.Run(cfg)

	if err != nil {
		t.Fatalf("Failed to start minimemcached: %v", err)
		return
	}

	defer mockMemcached.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	memcachedClient := NewMemcachedClient(logger, 120, "localhost:11212")

	testData := []byte("testData")
	initializer := func() (*Object, error) {
		return &Object{
			Key:  "testHost/testBucket/testKey",
			Data: &testData,
		}, nil
	}

	// Test case: Object is not in cache and needs to be initialized
	obj, err := memcachedClient.initialize("testKey", initializer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if obj == nil {
		t.Fatalf("Expected object, got nil")
	}
	if string(*obj.Data) != "testData" {
		t.Errorf("Expected data 'testData', got %s", string(*obj.Data))
	}

	// Test case: Object is already in cache
	obj, err = memcachedClient.initialize("testKey", initializer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(*obj.Data) != "testData" {
		t.Errorf("Expected data 'testData', got %s", string(*obj.Data))
	}

	// Test case: Initializer returns an error
	errorInitializer := func() (*Object, error) {
		return nil, ErrInitializer
	}
	_, err = memcachedClient.initialize("errorKey", errorInitializer)
	if err == nil || errors.Unwrap(err) != ErrInitializer {
		t.Errorf("Expected 'initializer error', got %v", err)
	}

	// Test case :Initializer is nil
	_, err = memcachedClient.initialize("nilKey", nil)
	if err == nil || err != ErrInitializerNil {
		t.Errorf("Expected 'initializer is nil', got %v", err)
	}
}

func compareObject(o1 Object, o2 Object) error {
	if o1.Key != o2.Key {
		return fmt.Errorf("Expected key %s, got %s", o2.Key, o1.Key)
	}
	if string(*o1.Data) != string(*o2.Data) {
		return fmt.Errorf("Expected data %s, got %s", string(*o2.Data), string(*o1.Data))
	}
	return nil
}

func TestPut(t *testing.T) {
	cfg := &minimemcached.Config{
		Port: 11212,
	}
	mockMemcached, err := minimemcached.Run(cfg)

	if err != nil {
		t.Fatalf("Failed to start minimemcached: %v", err)
		return
	}

	defer mockMemcached.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	memcachedClient := NewMemcachedClient(logger, 120, "localhost:11212")

	testData := []byte("testData")
	obj := &Object{
		Key:  "testHost/testBucket/testKey",
		Data: &testData,
	}

	err = memcachedClient.Put(obj)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockInitializer := func() (*Object, error) { return nil, fmt.Errorf("object not found") }

	// Test case: Object is not in cache
	cachedObj, err := memcachedClient.Get("testHost/testBucket/testKey", mockInitializer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	compareObject(*obj, *cachedObj)

	// Test case: Object is already in cache
	err = memcachedClient.Put(obj)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case: Object is nil
	err = memcachedClient.Put(nil)
	if err == nil || err != ErrObjectNil {
		t.Errorf("Expected 'object is nil', got %v", err)
	}

	//Test case: Object data is nil
	obj.Data = nil
	err = memcachedClient.Put(obj)
	if err == nil || err != ErrDataNil {
		t.Errorf("Expected 'object data is nil', got %v", err)
	}

	//Test case: memcached offline
	mockMemcached.Close()
	err = memcachedClient.Put(obj)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

}

func TestGet(t *testing.T) {
	cfg := &minimemcached.Config{
		Port: 11212,
	}
	mockMemcached, err := minimemcached.Run(cfg)

	if err != nil {
		t.Fatalf("Failed to start minimemcached: %v", err)
		return
	}

	defer mockMemcached.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	memcachedClient := NewMemcachedClient(logger, 120, "localhost:11212")

	testData := []byte("testData")
	obj := &Object{
		Key:  "testHost/testBucket/testKey",
		Data: &testData,
	}

	err = memcachedClient.Put(obj)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case: Object is in cache
	mockInitializer := func() (*Object, error) { return nil, fmt.Errorf("object not found") }
	cachedObj, err := memcachedClient.Get("testHost/testBucket/testKey", mockInitializer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	compareObject(*obj, *cachedObj)

	mockInitializer = func() (*Object, error) { return nil, ErrCacheMiss }

	// Test case: Object is not in cache
	_, err = memcachedClient.Get("testHost/testBucket/testKey2", mockInitializer)
	if err == nil || errors.Unwrap(err) != ErrCacheMiss {
		t.Errorf("Expected %v, got %v", ErrCacheMiss, err)
	}

	// Test case: Object is not in cache and initializer returns an error
	errorInitializer := func() (*Object, error) { return nil, ErrInitializer }
	_, err = memcachedClient.Get("testHost/testBucket/testKey2", errorInitializer)
	if err == nil || errors.Unwrap(err) != ErrInitializer {
		t.Errorf("Expected %v, got %v", ErrInitializer, errors.Unwrap(err))
	}

	// Test case: memcached offline
	mockMemcached.Close()
	_, err = memcachedClient.Get("testHost/testBucket/testKey", mockInitializer)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestDelete(t *testing.T) {
	cfg := &minimemcached.Config{
		Port: 11212,
	}
	mockMemcached, err := minimemcached.Run(cfg)

	if err != nil {
		t.Fatalf("Failed to start minimemcached: %v", err)
		return
	}

	defer mockMemcached.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	memcachedClient := NewMemcachedClient(logger, 120, "localhost:11212")

	testData := []byte("testData")
	obj := &Object{
		Key:  "testHost/testBucket/testKey",
		Data: &testData,
	}

	err = memcachedClient.Put(obj)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case: Object is in cache
	err = memcachedClient.Delete("testHost/testBucket/testKey")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case: Object is not in cache
	err = memcachedClient.Delete("testHost/testBucket/testKey")
	if err == nil || err != ErrCacheMiss {
		t.Fatalf("Expected %v, got %v", ErrCacheMiss, err)
	}

	// Test case: Invalid key
	err = memcachedClient.Delete("testHost/testBucket")
	if err == nil || err != ErrInvalidKey {
		t.Errorf("Expected %v, got %v", ErrInvalidKey, err)
	}

	//Test case: memcached offline
	mockMemcached.Close()
	err = memcachedClient.Delete("testHost/testBucket/testKey")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

// func TestTestConnection(t *testing.T) {
// 	cfg := &minimemcached.Config{
// 		Port: 11212,
// 	}
// 	mockMemcached, err := minimemcached.Run(cfg)

// 	if err != nil {
// 		t.Fatalf("Failed to start minimemcached: %v", err)
// 		return
// 	}

// 	defer mockMemcached.Close()

// 	logger := log.New(os.Stdout, "", log.LstdFlags)

// 	memcachedClient := NewMemcachedClient(logger, 120, "localhost:11212")

// 	// Test case: Connection is successful
// 	err = memcachedClient.TestConnection()
// 	if err != nil {
// 		t.Fatalf("Expected no error, got %v", err)
// 	}

// 	// Test case: Connection is unsuccessful
// 	mockMemcached.Close()
// 	time.Sleep(2 * time.Second)
// 	err = memcachedClient.TestConnection()
// 	if err == nil {
// 		t.Errorf("Expected error, got nil")
// 	}
// }
