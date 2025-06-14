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

func (a *AOF) Close() {
	a.file.Close()
}
