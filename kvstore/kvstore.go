package kvstore

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

const KeyNotFound = "ERROR: Key not found"
const DataFile = "data.txt"
const ExpirationsFile = "expirations.txt"

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
	value, exists := s.data[key]
	s.mutex.RUnlock()

	if !exists {
		return "", errors.New(KeyNotFound)
	}

	if s.expired(key) {
		s.mutex.Lock()
		delete(s.data, key)
		delete(s.expirations, key)
		s.mutex.Unlock()
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

func (s *KVStore) Delete(key string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, exists := s.data[key]
	if !exists {
		return errors.New(KeyNotFound)
	}
	delete(s.data, key)
	delete(s.expirations, key)
	return nil
}

func (s *KVStore) Keys() []string {
	s.cleanUp()

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

// Persistence Methods

func (s *KVStore) SaveToDisk(fileName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create file
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode data
	encoder := json.NewEncoder(file)
	return encoder.Encode(struct {
		Data        map[string]string
		Expirations map[string]time.Time
	}{
		Data:        s.data,
		Expirations: s.expirations,
	})
}

func (s *KVStore) LoadFromDisk(fileName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Open file
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode data
	var stored struct {
		Data        map[string]string
		Expirations map[string]time.Time
	}
	err = json.NewDecoder(file).Decode(&stored)
	if err != nil {
		return err
	}

	// Update in-memory storage
	s.data = stored.Data
	s.expirations = stored.Expirations
	return nil
}

// Helpers
func (s *KVStore) expired(key string) bool {
	exipration, exists := s.expirations[key]
	return exists && time.Now().After(exipration)
}

func (s *KVStore) cleanUp() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove expired keys
	for key, _ := range s.data {
		if s.expired(key) {
			s.Delete(key)
		}
	}
}
