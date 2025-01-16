package cache

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
)

// DummyPrinterCache is a dummy implementation of the Cache interface.
// It is used for testing purposes.

type DummyPrinterCache struct {
	logger  *log.Logger
	maxSize int64
	store   map[string]*Object
	lock    sync.RWMutex
}

func calculateKey(meta *ObjectMetadata) string {
	return meta.host + "/" + meta.bucket + "/" + meta.key
}

func printMetadata(meta *ObjectMetadata, logger *log.Logger) {
	if logger != nil {
		logger.Printf("Object:  %s/%s/%s", meta.host, meta.bucket, meta.key)
	}
}

func printObjectData(o *Object, logger *log.Logger) {
	if logger == nil {
		return
	}
	data, err := io.ReadAll(o.Reader)
	if err != nil {
		logger.Println("Error reading object data:", err)
		return
	}
	fmt.Println("Object Data:", string(data))
	o.Reader = bytes.NewReader(data) // Reset the reader to its original state
}

func (dpc *DummyPrinterCache) GetMetadata(key string) (*ObjectMetadata, error) {
	dpc.lock.RLock()
	defer dpc.lock.RUnlock()
	obj, exists := dpc.store[key]
	if !exists {
		return nil, fmt.Errorf("Object with key %s not found", key)
	}

	printMetadata(obj.Metadata, dpc.logger)

	return obj.Metadata, nil

}

func (dpc *DummyPrinterCache) Get(key string) (*Object, error) {
	dpc.lock.RLock()
	defer dpc.lock.RUnlock()
	obj, exists := dpc.store[key]
	if !exists {
		return nil, fmt.Errorf("Object with key %s not found", key)
	}

	dpc.logger.Println("Object retrieved from cache:")
	printMetadata(obj.Metadata, dpc.logger)
	printObjectData(obj, dpc.logger)

	return obj, nil
}

func (dpc *DummyPrinterCache) Put(o *Object) error {
	dpc.lock.Lock()
	defer dpc.lock.Unlock()
	var objData bytes.Buffer
	size, err := objData.ReadFrom(o)
	if err != nil {
		return err
	}
	if size > dpc.maxSize {
		return fmt.Errorf("Object size %d exceeds maximum size %d", size, dpc.maxSize)
	}
	o.Reader = bytes.NewReader(objData.Bytes())
	dpc.store[calculateKey(o.Metadata)] = o

	dpc.logger.Println("Object stored in cache:")
	printMetadata(o.Metadata, dpc.logger)
	printObjectData(o, dpc.logger)
	return nil

}

func NewDummyPrinterCache(logger *log.Logger, maxSize int64) *DummyPrinterCache {
	return &DummyPrinterCache{
		logger:  logger,
		maxSize: maxSize,
		store:   make(map[string]*Object),
	}
}
