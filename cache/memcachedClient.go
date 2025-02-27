package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedClient struct {
	client *memcache.Client
	ttl    int32
	logger *log.Logger
}

/*
Creates a new MemcachedClient with the given logger, default TTL and server addresses.
defaultTTL is expressed in seconds (max 1 month), or an absolute time in UNIX ecpoch.
*/
func NewMemcachedClient(logger *log.Logger, defaultTTL int32, server ...string) *MemcachedClient {
	return &MemcachedClient{
		client: memcache.New(server...),
		ttl:    defaultTTL,
		logger: logger,
	}
}

func (mw *MemcachedClient) Get(key string, initializer Initializer) (*Object, error) {

	obj, err := mw.get(key)

	if err != nil {

		return mw.initialize(key, initializer)

	}
	return obj, nil
}

func (mw *MemcachedClient) Put(obj *Object) error {
	return mw.set(obj)
}

func (mw *MemcachedClient) Delete(key string) error {

	if key == "" || len(strings.Split(key, "/")) < 3 {
		return ErrInvalidKey
	}
	err := mw.client.Delete(key)
	if err == memcache.ErrCacheMiss {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return nil
}

func (mw *MemcachedClient) Flush() error {
	return mw.client.FlushAll()
}

func (mw *MemcachedClient) TestConnection() error {
	return mw.client.Ping()
}

func (mw *MemcachedClient) set(obj *Object) error {

	if obj == nil {
		return ErrObjectNil
	}
	if obj.Key == "" {
		return ErrInvalidKey
	}
	if obj.Data == nil {
		return ErrDataNil
	}

	serialized, err := mw.serializeObj(*obj)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrSerialization, err)
	}

	return mw.client.Set(&memcache.Item{
		Key:        obj.Key,
		Value:      serialized,
		Expiration: mw.ttl,
	})
}

func (mw *MemcachedClient) get(key string) (*Object, error) {

	if key == "" || len(strings.Split(key, "/")) < 3 {
		return nil, ErrInvalidKey
	}

	serialized, err := mw.client.Get(key)

	if err != nil {
		return nil, err
	}

	obj, err := mw.deserializeObj(serialized.Value)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrDeserialization, err)
	}

	return &obj, nil
}

func (mw *MemcachedClient) initialize(key string, initializer Initializer) (*Object, error) {

	if initializer == nil {
		return nil, ErrInitializerNil
	}

	// double check if the object was not initialized by another goroutine
	obj, err := mw.get(key)

	if err != nil {

		// Initialize the object
		obj, err := initializer()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", ErrInitializer, err)
		}

		err = mw.set(obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (bw *MemcachedClient) serializeObj(o Object) ([]byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(o)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil

}

func (bw *MemcachedClient) deserializeObj(serialized []byte) (Object, error) {
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
