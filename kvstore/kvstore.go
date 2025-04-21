package kvstore

import (
	"errors"
	"sync"
)

const KeyNotFound = "ERROR: Key not found"

type KVStore struct {
	mutex sync.RWMutex
	data  map[string]string
}

func New() *KVStore {
	return &KVStore{
		data: make(map[string]string),
	}
}

func (s *KVStore) Set(key, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
}

func (s *KVStore) Get(key string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	value, exists := s.data[key]
	if !exists {
		return "", errors.New(KeyNotFound)
	}
	return value, nil
}
