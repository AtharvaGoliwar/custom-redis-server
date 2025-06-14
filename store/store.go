package store

import (
	"sync"
	"time"
)

// type Store struct {
// 	mu    sync.RWMutex
// 	store map[string]string
// }

type Item struct {
	Value      string
	Expiration int64 // Unix timestamp in seconds, 0 means no expiration
}

type Store struct {
	data map[string]Item
	mu   sync.RWMutex
}

func NewStore() *Store {
	s := &Store{
		data: make(map[string]Item),
	}
	// Start background cleaner
	go s.startCleaner()
	return s
}

func (s *Store) startCleaner() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now().Unix()
		for key, item := range s.data {
			if item.Expiration > 0 && now > item.Expiration {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Item{Value: value, Expiration: 0}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, found := s.data[key]
	if !found {
		return "", false
	}
	if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
		return "", false
	}
	return item.Value, true
}

func (s *Store) Del(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.data[key]; found {
		delete(s.data, key)
		return 1
	}
	return 0
}

func (s *Store) Exists(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, found := s.data[key]
	if !found {
		return 0
	}

	if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
		return 1
	}

	return 1
}

func (s *Store) Expire(key string, seconds int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, found := s.data[key]
	if !found {
		return false
	}

	if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
		return false
	}

	item.Expiration = time.Now().Unix() + seconds
	s.data[key] = item
	return true
}

func (s *Store) TTL(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, found := s.data[key]
	if !found {
		return -2 // Key does not exist
	}

	if item.Expiration == 0 {
		return -1 // Key has no expiration
	}

	remaining := item.Expiration - time.Now().Unix()
	if remaining <= 0 {
		return -2 // Already expired
	}

	return remaining
}

func (s *Store) Persist(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, found := s.data[key]
	if !found {
		return false
	}

	if item.Expiration == 0 {
		return false
	}

	item.Expiration = 0
	s.data[key] = item
	return true
}
