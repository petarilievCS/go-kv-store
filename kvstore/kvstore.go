package kvstore

import (
	"encoding/json"
	"errors"
	"log"
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

func (s *KVStore) Contains(key string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.data[key]
	return exists
}

func (s *KVStore) SetEx(key string, value string, ttl int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
	s.expirations[key] = time.Now().Add(time.Duration(ttl) * time.Second)
}

func (s *KVStore) TTL(key string) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.data[key]
	if !exists {
		return -2
	}

	ttl, exists := s.expirations[key]
	if !exists {
		return -1
	}

	secondsRemaining := int(time.Until(ttl).Seconds())
	if secondsRemaining < 0 {
		return -2
	}

	return secondsRemaining
}

func (s *KVStore) Persist(key string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, keyExists := s.data[key]
	if !keyExists {
		return 0
	}

	_, expirationExists := s.expirations[key]
	if !expirationExists {
		return 0
	}

	delete(s.expirations, key)
	return 1
}

func (s *KVStore) Rename(oldKey string, newKey string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, exists := s.data[oldKey]
	if !exists {
		return 0
	}

	delete(s.data, oldKey)
	s.data[newKey] = value

	expiration, hasExpiration := s.expirations[oldKey]
	if hasExpiration {
		delete(s.expirations, oldKey)
		s.expirations[newKey] = expiration
	}
	return 1
}

func (s *KVStore) RenameNX(oldKey string, newKey string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, exists := s.data[oldKey]
	if !exists {
		return 0
	}

	_, newKeyExists := s.data[newKey]
	if newKeyExists {
		return 0
	}

	delete(s.data, oldKey)
	s.data[newKey] = value

	expiration, hasExpiration := s.expirations[oldKey]
	if hasExpiration {
		delete(s.expirations, oldKey)
		s.expirations[newKey] = expiration
	}
	return 1
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

func (s *KVStore) Flush() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data = make(map[string]string)
	s.expirations = make(map[string]time.Time)
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

func (s *KVStore) KeysWithTTL() []string {
	s.cleanUp()

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	keys := make([]string, 0, len(s.expirations))
	for key := range s.expirations {
		keys = append(keys, key)
	}
	return keys
}

func (s *KVStore) KeysNoTTL() []string {
	s.cleanUp()

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var keys []string
	for key := range s.data {
		_, hasExpiration := s.expirations[key]
		if !hasExpiration {
			keys = append(keys, key)
		}
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
			delete(s.data, key)
			delete(s.expirations, key)
		}
	}
}

func (s *KVStore) ScheduleCleanup(interval time.Duration, done <-chan struct{}) {
	log.Printf("[INFO] Scheduled cleanup every %v seconds\n", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("[INFO] Running scheduled cleanup...")
				s.cleanUp()
			case <-done:
				log.Println("[INFO] Stopping scheduled cleanup...")
				return
			}
		}
	}()
}
