package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
)

type StatsLogEntry struct {
	Timestamp time.Time
	bigcache.Stats
}

type StatsLog struct {
	Entries []StatsLogEntry
	sync.Mutex
}

type BigcacheWrapper struct {
	bc     *bigcache.BigCache
	logger *log.Logger
	stats  StatsLog
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
	config := bigcache.Config{
		Shards:             32,
		LifeWindow:         10 * time.Minute,
		CleanWindow:        1 * time.Second,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       1000000,
		StatsEnabled:       false,
		Verbose:            true,
		HardMaxCacheSize:   512,
		Logger:             logger,
	}

	bc, err := bigcache.New(context.Background(), config)
	if err != nil {
		logger.Fatalf("Error creating bigcache instance: %v", err)
	}

	return &BigcacheWrapper{
		bc:     bc,
		logger: logger,
	}
}

func (bw *BigcacheWrapper) Get(key string) (*Object, error) {
	data, err := bw.get(key)

	if err != nil {
		//object not found
		if err == bigcache.ErrEntryNotFound {

			return nil, ErrCacheMiss
		}
		return nil, err
	}
	return data, nil
}

func (bw *BigcacheWrapper) GetTimed(key string, initializer Initializer) (*Object, int64, int64, error) {
	start := time.Now()
	data, err := bw.get(key)
	elapsed := time.Since(start).Nanoseconds()

	if err != nil {
		//object not found
		start := time.Now()
		obj, err := bw.initialize(key, initializer)
		initElapsed := time.Since(start).Nanoseconds()
		return obj, elapsed, initElapsed, err
	}
	return data, elapsed, 0, err
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
		go bw.put(obj)
		return obj, nil
	}

	return obj, nil
}

func (bw *BigcacheWrapper) put(o *Object) error {
	serializedObj, err := bw.serializeObj(*o)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrSerialization, err)
	}
	err = bw.bc.Set(o.Key, serializedObj)
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

func (bw *BigcacheWrapper) GetStats() *StatsLog {
	return &bw.stats
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

func (bw *BigcacheWrapper) SaveStats() {
	stats := bw.bc.Stats()

	bw.stats.Lock()
	bw.stats.Entries = append(bw.stats.Entries, StatsLogEntry{
		Timestamp: time.Now(),
		Stats:     stats,
	})
	bw.stats.Unlock()

}

func (sl *StatsLog) WriteCSV(f *os.File) error {
	sl.Lock()
	defer sl.Unlock()
	_, err := f.WriteString("time,hits,misses,delete_hits,delete_misses,collisions\n")

	if err != nil {
		return err
	}

	for _, s := range sl.Entries {
		_, err := f.WriteString(fmt.Sprintf("%v,%d,%d,%d,%d,%d\n", s.Timestamp.Format(time.RFC3339), s.Hits, s.Misses, s.DelHits, s.DelMisses, s.Collisions))
		if err != nil {
			return err
		}
	}
	return nil
}
