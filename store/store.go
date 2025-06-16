package store

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	Data map[string]Item
	Mu   sync.RWMutex
}

func NewStore() *Store {
	s := &Store{
		Data: make(map[string]Item),
	}
	// Start background cleaner
	go s.startCleaner()
	return s
}

func (s *Store) startCleaner() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.Mu.Lock()
		now := time.Now().Unix()
		for key, item := range s.Data {
			if item.Expiration > 0 && now > item.Expiration {
				delete(s.Data, key)
			}
		}
		s.Mu.Unlock()
	}
}

func (s *Store) Set(key, value string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Data[key] = Item{Value: value, Expiration: 0}
}

func (s *Store) Get(key string) (string, bool) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	item, found := s.Data[key]
	if !found {
		return "", false
	}
	return item.Value, true
}

func (s *Store) Del(key string) int {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if _, found := s.Data[key]; found {
		delete(s.Data, key)
		return 1
	}
	return 0
}

func (s *Store) Exists(key string) int {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	item, found := s.Data[key]
	if !found {
		return 0
	}

	if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
		return 1
	}

	return 1
}

func (s *Store) Expire(key string, seconds int64) bool {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	item, found := s.Data[key]
	if !found {
		return false
	}

	if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
		return false
	}

	item.Expiration = time.Now().Unix() + seconds
	s.Data[key] = item
	return true
}

func (s *Store) TTL(key string) int64 {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	item, found := s.Data[key]
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
	s.Mu.Lock()
	defer s.Mu.Unlock()

	item, found := s.Data[key]
	if !found {
		return false
	}

	if item.Expiration == 0 {
		return false
	}

	item.Expiration = 0
	s.Data[key] = item
	return true
}

// SaveRDB writes the entire database to a dump.rdb file
func (s *Store) SaveRDB(filename string) error {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for key, item := range s.Data {
		// Serialize as: key|value|expiration\n
		line := fmt.Sprintf("%s|%s|%d\n", key, item.Value, item.Expiration)
		_, err := writer.WriteString(line)
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

// LoadRDB reads the dump.rdb file and populates the store
func (s *Store) LoadRDB(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// It's fine if file doesn't exist (first run)
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	s.Mu.Lock()
	defer s.Mu.Unlock()

	now := time.Now().Unix()

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		key := parts[0]
		value := parts[1]
		expiration, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			expiration = 0
		}

		// Check if expired
		if expiration != 0 && expiration <= now {
			continue // skip expired keys
		}

		s.Data[key] = Item{
			Value:      value,
			Expiration: expiration,
		}
	}

	return scanner.Err()
}
