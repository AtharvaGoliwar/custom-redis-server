package client

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Client represents a simple RESP client connection
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewClient creates a new RESP client
func NewClient(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, nil
}

// Close closes the connection
func (c *Client) Close() {
	c.conn.Close()
}

// Send sends a command (args) in RESP format
func (c *Client) Send(args ...string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	_, err := c.writer.WriteString(sb.String())
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

// Receive reads and parses server response
func (c *Client) Receive() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", nil
	}

	switch line[0] {
	case '+', '-', ':':
		return line[1:], nil

	case '$':
		length, err := strconv.Atoi(line[1:])
		if err != nil || length < 0 {
			return "(nil)", nil
		}

		buf := make([]byte, length+2) // include \r\n
		_, err = c.reader.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:length]), nil

	default:
		return line, nil
	}
}
