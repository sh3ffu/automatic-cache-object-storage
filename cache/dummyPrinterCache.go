package cache

import (
	"fmt"
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

// func printObjectData(o *Object, logger *log.Logger) {
// 	if logger == nil {
// 		return
// 	}
// 	data, err := io.ReadAll(bytes.NewReader(*o.Data))
// 	if err != nil {
// 		logger.Println("Error reading object data:", err)
// 		return
// 	}
// 	fmt.Println("Object Data:", string(data))
// }

func (dpc *DummyPrinterCache) Get(key string, initializer Initializer) (*Object, error) {
	obj, exists := dpc.get(key)
	if !exists {
		//dpc.logger.Println("Attempting to retrieve object from remote:")
		return dpc.initialize(key, initializer)
	}

	//dpc.logger.Println("Object retrieved from cache:")
	//printMetadata(obj.Metadata, dpc.logger)
	//go printAction("Get", obj.Metadata, dpc.logger)

	return obj, nil
}

func (dpc *DummyPrinterCache) Put(o *Object) error {
	dpc.lock.Lock()
	defer dpc.lock.Unlock()
	dpc.store[o.Key] = o

	// dpc.logger.Println("Object stored in cache:")
	// go printMetadata(o.Metadata, dpc.logger)
	// printObjectData(o, dpc.logger)

	//go printAction("Put", o.Metadata, dpc.logger)

	return nil

}

func (dpc *DummyPrinterCache) get(key string) (*Object, bool) {
	dpc.lock.RLock()
	defer dpc.lock.RUnlock()
	obj, exists := dpc.store[key]
	return obj, exists
}

func (dpc *DummyPrinterCache) initialize(key string, initializer Initializer) (*Object, error) {

	data, exists := dpc.get(key)
	if exists {
		return data, nil
	}

	if initializer != nil {

		obj, err := initializer()
		if err != nil {
			return nil, err
		}
		dpc.Put(obj)
		return obj, nil

	}
	return nil, fmt.Errorf("Object with key %s not found", key)
}

func NewDummyPrinterCache(logger *log.Logger, maxSize int64) *DummyPrinterCache {
	return &DummyPrinterCache{
		logger:  logger,
		maxSize: maxSize,
		store:   make(map[string]*Object, 400),
	}
}
