package server

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"redis-server/protocol"
	"redis-server/store"
)

type Server struct {
	Store *store.Store
}

type ClientState struct {
	InTransaction    bool
	TransactionQueue [][]string
}

func NewServer() *Server {
	return &Server{
		Store: store.NewStore(),
	}
}

func (s *Server) Start(address string) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Printf("Server started on %s\n", address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	clientState := &ClientState{}

	for {
		tokens, err := protocol.Parse(reader)
		if err != nil {
			conn.Write([]byte(protocol.EncodeError("invalid request")))
		}

		cmd := strings.ToUpper(tokens[0])

		if len(tokens) == 0 {
			continue
		}
		// MULTI
		if cmd == "MULTI" {
			clientState.InTransaction = true
			clientState.TransactionQueue = [][]string{}
			conn.Write([]byte(protocol.EncodeSimple("OK")))
			continue
		}

		// DISCARD
		if cmd == "DISCARD" {
			clientState.InTransaction = false
			clientState.TransactionQueue = [][]string{}
			conn.Write([]byte(protocol.EncodeSimple("OK")))
			continue
		}

		// Transaction Queuing
		if clientState.InTransaction && cmd != "EXEC" {
			clientState.TransactionQueue = append(clientState.TransactionQueue, tokens)
			conn.Write([]byte(protocol.EncodeSimple("QUEUED")))
			continue
		}

		// EXEC
		if cmd == "EXEC" {
			if !clientState.InTransaction {
				conn.Write([]byte(protocol.EncodeError("ERR EXEC without MULTI")))
				continue
			}
			// Validate transaction first
			aborted := false
			for _, queued := range clientState.TransactionQueue {
				if !isValidCommand(queued) {
					conn.Write([]byte(protocol.EncodeError("Transaction aborted: invalid command " + queued[0])))
					clientState.InTransaction = false
					clientState.TransactionQueue = [][]string{}
					aborted = true
					break
				}
			}
			if aborted {
				continue
			}

			// Execute
			conn.Write([]byte(protocol.EncodeArrayHeader(len(clientState.TransactionQueue))))
			for _, queued := range clientState.TransactionQueue {
				reply := s.executeCommand(queued)
				conn.Write([]byte(reply))
			}

			clientState.InTransaction = false
			clientState.TransactionQueue = [][]string{}
			continue
		}

		// Normal command path
		reply := s.executeCommand(tokens)
		conn.Write([]byte(reply))
	}
}

func isValidCommand(tokens []string) bool {
	cmd := strings.ToUpper(tokens[0])

	switch cmd {
	case "EXPIRE":
		return len(tokens) == 3
	case "PERSIST", "TTL":
		return len(tokens) == 2
	case "DEL", "EXISTS":
		return len(tokens) >= 2
	case "SET":
		return (len(tokens)-1)%2 == 0
	case "GET":
		return len(tokens) >= 2
	default:
		return false
	}
}

func (s *Server) executeCommand(tokens []string) string {
	cmd := strings.ToUpper(tokens[0])
	switch cmd {
	case "DEL":
		if len(tokens) < 2 {
			return protocol.EncodeError("wrong number of arguments for DEL")
		}
		count := 0
		for _, key := range tokens[1:] {
			count += s.Store.Del(key)
		}
		return protocol.EncodeInteger(count)
	case "EXISTS":
		if len(tokens) < 2 {
			return protocol.EncodeError("wrong number of arguments for DEL")
		}
		count := 0
		for _, key := range tokens[1:] {
			count += s.Store.Exists(key)
		}
		return protocol.EncodeInteger(count)
	case "SET":
		if len(tokens)%2 != 1 {
			return protocol.EncodeError("wrong number of arguments for SET")
		}
		for i := 1; i < len(tokens); i += 2 {
			s.Store.Set(tokens[i], tokens[i+1])
		}
		return protocol.EncodeSimple("OK")
	case "GET":
		if len(tokens) < 2 {
			return protocol.EncodeError("wrong number of arguments for GET")
		}
		if len(tokens) > 2 {
			values := []string{}
			for i := 1; i < len(tokens); i++ {
				val, ok := s.Store.Get(tokens[i])
				if !ok {
					values = append(values, protocol.EncodeBulk("(nil)"))
				} else {
					values = append(values, protocol.EncodeBulk(val))
				}
			}
			return protocol.EncodeArrayRaw(values)
		} else {
			val, ok := s.Store.Get(tokens[1])
			if !ok {
				return protocol.EncodeBulk("(nil)")
			}
			return protocol.EncodeBulk(val)
		}

	case "EXPIRE":
		if len(tokens) != 3 {
			return protocol.EncodeError("wrong number of arguments for EXPIRE")
		}
		seconds, err := strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			return protocol.EncodeError("invalid seconds argument")
		}
		success := s.Store.Expire(tokens[1], seconds)
		if success {
			return protocol.EncodeInteger(1)
		} else {
			return protocol.EncodeInteger(0)
		}

	case "TTL":
		if len(tokens) != 2 {
			return protocol.EncodeError("wrong number of arguments for TTL")
		}
		ttl := s.Store.TTL(tokens[1])
		return protocol.EncodeInteger(int(ttl))

	case "PERSIST":
		if len(tokens) != 2 {
			return protocol.EncodeError("wrong number of arguments for PERSIST")
		}
		success := s.Store.Persist(tokens[1])
		if success {
			return protocol.EncodeInteger(1)
		} else {
			return protocol.EncodeInteger(0)
		}

	case "QUIT":
		return protocol.EncodeSimple("BYE!")

	default:
		return protocol.EncodeError("Unknown command")
	}
}
