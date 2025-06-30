package fredisdb

import (
	"log"
	"os"
)

type AOF struct {
	file *os.File
}

func NewAOF(path string) *AOF {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open AOF log file: %v", err)
	}
	return &AOF{file: file}
}

func (a *AOF) LogCommand(cmd string) {
	_, err := a.file.WriteString(cmd)
	if err != nil {
		log.Printf("AOF write error: %v", err)
	}
}

func (a *AOF) Close() {
	a.file.Close()
}
