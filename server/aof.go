package server

import (
	"fmt"
	"os"
)

type AOF struct {
	file *os.File
}

func NewAOF(filename string) (*AOF, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &AOF{file: file}, nil
}

func (a *AOF) Append(command []string) error {
	line := fmt.Sprintf("*%d\r\n", len(command))
	for _, arg := range command {
		line += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}
	_, err := a.file.WriteString(line)
	return err
}

// func (a *AOF) Rewrite(store *Store) error {
// 	tmpFile, err := os.Create("appendonly.aof.tmp")
// 	if err != nil {
// 		return err
// 	}
// 	defer tmpFile.Close()

// 	store.mu.RLock()
// 	defer store.mu.RUnlock()

// 	for key, item := range store.data {
// 		// Skip expired keys
// 		if item.Expiration > 0 && time.Now().Unix() > item.Expiration {
// 			continue
// 		}

// 		// Write SET command
// 		line := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
// 			len(key), key, len(item.Value), item.Value)
// 		_, err := tmpFile.WriteString(line)
// 		if err != nil {
// 			return err
// 		}

// 		// Write EXPIRE command if TTL exists
// 		if item.Expiration > 0 {
// 			ttl := item.Expiration - time.Now().Unix()
// 			if ttl > 0 {
// 				expireLine := fmt.Sprintf("*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n",
// 					len(key), key, len(fmt.Sprintf("%d", ttl)), ttl)
// 				_, err := tmpFile.WriteString(expireLine)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	// Atomically replace original AOF
// 	err = os.Rename("appendonly.aof.tmp", "appendonly.aof")
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (a *AOF) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}
func (a *AOF) Reopen(filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.file = file
	return nil
}
