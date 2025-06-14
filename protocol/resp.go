package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Parse reads and parses RESP array commands (like SET foo bar)
func Parse(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if len(line) == 0 || line[0] != '*' {
		return nil, errors.New("invalid RESP array")
	}

	numElements, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, errors.New("invalid array count")
	}

	parts := make([]string, 0, numElements)

	for i := 0; i < numElements; i++ {
		// Read bulk string header: $<length>
		prefix, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if len(prefix) == 0 || prefix[0] != '$' {
			return nil, errors.New("invalid bulk string")
		}

		strLen, err := strconv.Atoi(strings.TrimSpace(prefix[1:]))
		if err != nil {
			return nil, err
		}

		// Read actual string data
		buf := make([]byte, strLen+2) // +2 for \r\n
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}

		parts = append(parts, string(buf[:strLen]))
	}

	return parts, nil
}

// Encode simple string response
func EncodeSimple(s string) string {
	return fmt.Sprintf("+%s\r\n", s)
}

// Encode error response
func EncodeError(s string) string {
	return fmt.Sprintf("-%s\r\n", s)
}

// Encode bulk string
func EncodeBulk(s string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func EncodeInteger(i int) string {
	return fmt.Sprintf(":%d\r\n", i)
}

func EncodeArrayRaw(items []string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("*%d\r\n", len(items)))
	for _, item := range items {
		result.WriteString(item)
	}
	return result.String()
}

func EncodeArrayHeader(n int) string {
	return fmt.Sprintf("*%d\r\n", n)
}
