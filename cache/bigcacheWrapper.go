package cache

import (
	"bytes"
	"context"
	"encoding/gob"
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

	if initializer == nil {
		return nil, ErrInitializerNil
	}

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
	serializedObj, err := bw.serializeObj(*o)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrSerialization, err)
	}
	err = bw.bc.Set(fmt.Sprintf("%s/%s/%s", o.Metadata.Host, o.Metadata.Bucket, o.Metadata.Key), serializedObj)
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

	o, err := bw.deserializeObj(data)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrDeserialization, err)
	}

	return &o, nil
}

func (bw *BigcacheWrapper) GetStats() string {
	return fmt.Sprintf("Bigcache stats: Hits: %d, Misses: %d, DelHits: %d, DelMisses: %d, Collisions: %d", bw.bc.Stats().Hits, bw.bc.Stats().Misses, bw.bc.Stats().DelHits, bw.bc.Stats().DelMisses, bw.bc.Stats().Collisions)
}

func (bw *BigcacheWrapper) serializeObj(o Object) ([]byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(o)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil

}

func (bw *BigcacheWrapper) deserializeObj(serialized []byte) (Object, error) {
	o := Object{}
	b := bytes.Buffer{}
	b.Write(serialized)
	d := gob.NewDecoder(&b)
	err := d.Decode(&o)
	if err != nil {
		return Object{}, err
	}
	return o, nil
}
