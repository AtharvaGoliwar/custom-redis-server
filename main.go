// package main

// import (
// 	"bufio"
// 	"fmt"
// 	"net"
// 	"strings"
// 	"sync"
// )

// // Simple in-memory store with mutex for thread safety
// type Store struct {
// 	mu    sync.RWMutex
// 	store map[string]string
// }

// func NewStore() *Store {
// 	return &Store{
// 		store: make(map[string]string),
// 	}
// }

// func (s *Store) Set(key, value string) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.store[key] = value
// }

// func (s *Store) Get(key string) (string, bool) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	val, ok := s.store[key]
// 	return val, ok
// }

// // Command handler
// func handleConnection(conn net.Conn, store *Store) {
// 	defer conn.Close()

// 	reader := bufio.NewReader(conn)
// 	for {
// 		conn.Write([]byte("> "))
// 		input, err := reader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Connection closed.")
// 			return
// 		}

// 		input = strings.TrimSpace(input)
// 		tokens := strings.SplitN(input, " ", 3)
// 		if len(tokens) == 0 {
// 			continue
// 		}

// 		command := strings.ToUpper(tokens[0])

// 		switch command {
// 		case "SET":
// 			if len(tokens) != 3 {
// 				conn.Write([]byte("ERR Usage: SET key value\n"))
// 				continue
// 			}
// 			store.Set(tokens[1], tokens[2])
// 			conn.Write([]byte("OK\n"))
// 		case "GET":
// 			if len(tokens) != 2 {
// 				conn.Write([]byte("ERR Usage: GET key\n"))
// 				continue
// 			}
// 			if val, ok := store.Get(tokens[1]); ok {
// 				conn.Write([]byte(val + "\n"))
// 			} else {
// 				conn.Write([]byte("(nil)\n"))
// 			}
// 		case "QUIT":
// 			conn.Write([]byte("Bye!\n"))
// 			return
// 		default:
// 			conn.Write([]byte("ERR unknown command\n"))
// 		}
// 	}
// }

// func main() {
// 	listener, err := net.Listen("tcp", ":6379") // redis default port
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer listener.Close()

// 	fmt.Println("Custom Redis Server running on port 6379...")

// 	store := NewStore()

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			fmt.Println("Error accepting:", err)
// 			continue
// 		}
// 		go handleConnection(conn, store) // handle each client in goroutine
// 	}
// }

package main

import "redis-server/server"

func main() {
	s := server.NewServer()
	s.Start(":6380")
}
