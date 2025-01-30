package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/allegro/bigcache/v3"
)

type BigcacheWrapper struct {
	bc     *bigcache.BigCache
	logger *log.Logger
}

/*
NewBigcacheWrapper creates a new instance of the BigcacheWrapper
*/
func NewBigcacheWrapper(logger *log.Logger, maxMemory int) *BigcacheWrapper {

	// config := bigcache.Config{
	// 	Shards:             1024,
	// 	LifeWindow:         10,
	// 	CleanWindow:        10,
	// 	MaxEntriesInWindow: 1000 * 10 * 60,
	// 	MaxEntrySize:       500000,
	// 	Verbose:            true,
	// 	HardMaxCacheSize:   maxMemory,
	// 	Logger:             logger,
	// 	OnRemove:           nil,
	// 	OnRemoveWithReason: nil,
	// }

	bc, err := bigcache.New(context.Background(), bigcache.DefaultConfig(10*time.Minute))
	if err != nil {
		logger.Fatalf("Error creating bigcache instance: %v", err)
	}

	return &BigcacheWrapper{
		bc:     bc,
		logger: logger,
	}
}

func (bw *BigcacheWrapper) Get(key string, initializer Initializer) (*Object, error) {
	data, err := bw.get(key)

	if err != nil {
		//object not found

		return bw.initialize(key, initializer)
	}

	return data, nil
}

func (bw *BigcacheWrapper) GetMetadata(key string) (*ObjectMetadata, error) {
	_, err := bw.get(key)
	if err != nil {
		return nil, err
	}
	return NewMetadata(key)
}

func (bw *BigcacheWrapper) Put(o *Object) error {
	return bw.put(o)
}

func (bw *BigcacheWrapper) initialize(key string, initializer Initializer) (*Object, error) {

	// double check if the object was not initialized by another goroutine
	obj, err := bw.get(key)

	if err != nil {
		obj, err := initializer()
		if err != nil {
			return nil, err
		}
		err = bw.put(obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (bw *BigcacheWrapper) put(o *Object) error {
	err := bw.bc.Set(fmt.Sprintf("%s/%s/%s", o.Metadata.Host, o.Metadata.Bucket, o.Metadata.Key), *o.Data)
	if err != nil {
		return err
	}
	return nil
}

func (bw *BigcacheWrapper) get(key string) (*Object, error) {
	data, err := bw.bc.Get(key)
	if err != nil {
		return nil, err
	}

	meta, err := NewMetadata(key)
	if err != nil {
		return nil, err
	}

	return &Object{Data: &data, Metadata: meta}, nil
}

func (bw *BigcacheWrapper) GetStats() string {
	return fmt.Sprintf("Bigcache stats: Hits: %d, Misses: %d, DelHits: %d, DelMisses: %d, Collisions: %d", bw.bc.Stats().Hits, bw.bc.Stats().Misses, bw.bc.Stats().DelHits, bw.bc.Stats().DelMisses, bw.bc.Stats().Collisions)
}
