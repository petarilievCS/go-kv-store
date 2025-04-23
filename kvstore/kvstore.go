package kvstore

import (
	"errors"
	"sync"
	"time"
)

const KeyNotFound = "ERROR: Key not found"

type KVStore struct {
	mutex       sync.RWMutex
	data        map[string]string
	expirations map[string]time.Time
}

func New() *KVStore {
	return &KVStore{
		data:        make(map[string]string),
		expirations: make(map[string]time.Time),
	}
}

func (s *KVStore) Set(key, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value

	_, exists := s.expirations[key]
	if exists {
		delete(s.expirations, key)
	}
}

func (s *KVStore) Get(key string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	value, exists := s.data[key]
	if !exists {
		return "", errors.New(KeyNotFound)
	}
	if s.expired(key) {
		delete(s.data, key)
		delete(s.expirations, key)
		return "", errors.New(KeyNotFound)
	}
	return value, nil
}

func (s *KVStore) SetEx(key string, value string, ttl int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
	s.expirations[key] = time.Now().Add(time.Duration(ttl) * time.Second)
}

func (s *KVStore) expired(key string) bool {
	exipration, exists := s.expirations[key]
	return exists && time.Now().After(exipration)
}
