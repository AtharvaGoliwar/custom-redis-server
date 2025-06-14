package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"redis-server/protocol"
	"redis-server/store"
)

type Server struct {
	Store *store.Store
	aof   *AOF
	ln    net.Listener // hold listener here for shutdown

	credentials map[string]string // username -> password
	userGroups  map[string]string // username -> group

	clients sync.Map // map[net.Conn]*ClientSession
}

type ClientState struct {
	InTransaction    bool
	TransactionQueue [][]string
}

type ClientSession struct {
	authenticated bool
	username      string
	group         string
}

func NewServer() *Server {
	aof, err := NewAOF("appendonly.aof")
	if err != nil {
		log.Fatal("AOF error: ", err)
	}
	server := &Server{
		Store: store.NewStore(),
		aof:   aof,
		credentials: map[string]string{
			"user1": "pass1",
			"user2": "pass2",
			"admin": "specialpassword",
		},
		userGroups: map[string]string{
			"user1": "group1",
			"user2": "group2",
			"admin": "admin",
		},
	}
	server.ReplayAOF()
	return server
	// return &Server{
	// 	Store: store.NewStore(),
	// 	aof:   aof,
	// }
}

func (s *Server) ReplayAOF() {
	file, err := os.Open("appendonly.aof")
	if err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		// tokens, err := protocol.Parse("", reader)
		tokens, err := protocol.Parse(reader)
		if err != nil {
			break
		}
		s.executeCommand(tokens, nil, nil) // no need to re-append during replay
	}
}

func (s *Server) Start(address string) {
	var err error
	s.ln, err = net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer s.ln.Close()

	fmt.Printf("Server started on %s\n", address)

	go s.handleShutdown()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleShutdown() {
	// Create channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop // Block until signal received

	fmt.Println("\nShutting down server...")

	s.ln.Close()  // Close TCP listener
	s.aof.Close() // Close AOF file

	fmt.Println("Server gracefully stopped.")
	os.Exit(0)
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clients.Delete(conn)
	}()

	reader := bufio.NewReader(conn)
	clientState := &ClientState{}

	session := &ClientSession{}
	s.clients.Store(conn, session)

	for {
		tokens, err := protocol.Parse(reader)
		if err != nil {
			conn.Write([]byte(protocol.EncodeError("invalid request")))
		}

		if len(tokens) == 0 {
			continue
		}
		cmd := strings.ToUpper(tokens[0])
		// AUTH
		if cmd == "AUTH" {
			if len(tokens) != 3 {
				conn.Write([]byte(protocol.EncodeError("Wrong number of arguments for AUTH")))
				continue
			}
			username, password := tokens[1], tokens[2]
			if s.credentials[username] == password {
				session.authenticated = true
				session.username = username
				session.group = s.userGroups[username]
				conn.Write([]byte(protocol.EncodeSimple("OK")))
			} else {
				conn.Write([]byte(protocol.EncodeError("Invalid credentials")))
			}
			continue
		}

		if !session.authenticated {
			conn.Write([]byte(protocol.EncodeError("NOAUTH Authentication required")))
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
				reply := s.executeCommand(queued, s.aof, session)
				conn.Write([]byte(reply))
			}

			clientState.InTransaction = false
			clientState.TransactionQueue = [][]string{}
			continue
		}

		// Normal command path
		reply := s.executeCommand(tokens, s.aof, session)
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
	case "ME":
		return len(tokens) == 1
	case "ADDUSER":
		return len(tokens) == 4
	case "DELUSER":
		return len(tokens) == 2
	default:
		return false
	}
}

func (s *Server) executeCommand(tokens []string, aof *AOF, session *ClientSession) string {
	cmd := strings.ToUpper(tokens[0])

	// ADDUSER
	if cmd == "ADDUSER" {
		if session != nil {
			if session.group != "admin" && aof != nil {
				return protocol.EncodeError("Permission denied")
			}
		}
		if len(tokens) != 4 {
			return protocol.EncodeError("Usage: ADDUSER <username> <password> <group>")
		}
		username, password, group := tokens[1], tokens[2], tokens[3]
		s.credentials[username] = password
		s.userGroups[username] = group
		if aof != nil {
			aof.Append(tokens)
		}
		return protocol.EncodeSimple("OK")
	}

	// DELUSER
	if cmd == "DELUSER" {
		if session != nil {
			if session.group != "admin" && aof != nil {
				return protocol.EncodeError("Permission denied")
			}
		}
		if len(tokens) != 2 {
			return protocol.EncodeError("Usage: DELUSER <username>")
		}
		username := tokens[1]
		delete(s.credentials, username)
		delete(s.userGroups, username)
		if aof != nil {
			aof.Append(tokens)
		}
		return protocol.EncodeSimple("OK")
	}

	// LISTUSERS (newly added)
	if cmd == "LISTUSERS" {
		if session.group != "admin" {
			return protocol.EncodeError("Permission denied")
		}
		return s.listUsers()
	}

	if cmd == "INFO" && len(tokens) > 1 && tokens[1] == "USERS" && session.group == "admin" {
		return s.getActiveUsers()
	}

	// Enforce key access for commands that involve keys
	if session != nil && session.group != "admin" && len(tokens) > 1 {
		key := tokens[1]
		if !strings.HasPrefix(key, session.group+":") {
			return protocol.EncodeError("Permission denied for key " + key)
		}
	}

	switch cmd {
	case "DEL":
		if len(tokens) < 2 {
			return protocol.EncodeError("wrong number of arguments for DEL")
		}
		count := 0
		for _, key := range tokens[1:] {
			count += s.Store.Del(key)
		}
		if aof != nil {
			aof.Append(tokens)
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
		if aof != nil {
			aof.Append(tokens)
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
			if aof != nil {
				aof.Append(tokens)
			}
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
			if aof != nil {
				aof.Append(tokens)
			}
			return protocol.EncodeInteger(1)
		} else {
			return protocol.EncodeInteger(0)
		}

	case "ME":
		values := []string{}
		if len(tokens) > 1 {
			return protocol.EncodeError("wrong number of arguments for ME")
		}
		values = append(values, protocol.EncodeBulk(session.username))
		values = append(values, protocol.EncodeBulk(session.group))
		return protocol.EncodeArrayRaw(values)

	case "QUIT":
		return protocol.EncodeSimple("BYE!")

	default:
		return protocol.EncodeError("Unknown command")
	}
}

func (s *Server) getActiveUsers() string {
	var result strings.Builder
	s.clients.Range(func(key, value interface{}) bool {
		client := value.(*ClientSession)
		if client.authenticated {
			result.WriteString(fmt.Sprintf("user=%s group=%s\n", client.username, client.group))
		}
		return true
	})

	return protocol.EncodeBulk(result.String())
}

func (s *Server) listUsers() string {
	parts := []string{}
	for username, group := range s.userGroups {
		parts = append(parts, fmt.Sprintf("user=%s group=%s", username, group))
	}
	resp := protocol.EncodeArray(parts)
	// fmt.Println("LISTUSERS RESP:", resp)
	return resp

}
