package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type TestCase struct {
	Name     string
	Command  string
	Expected string // Raw RESP expected response
}

func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	tests := []TestCase{
		{
			Name:     "PING simple",
			Command:  "*1\r\n$4\r\nPING\r\n",
			Expected: "+PONG\r\n",
		},
		{
			Name:     "PING with message",
			Command:  "*2\r\n$4\r\nPING\r\n$5\r\nhello\r\n",
			Expected: "$5\r\nhello\r\n",
		},
		{
			Name:     "SET key",
			Command:  "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			Expected: "+OK\r\n",
		},
		{
			Name:     "GET key",
			Command:  "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n",
			Expected: "$3\r\nbar\r\n",
		},
		{
			Name:     "DEL key",
			Command:  "*2\r\n$3\r\nDEL\r\n$3\r\nfoo\r\n",
			Expected: ":1\r\n",
		},
		{
			Name:     "GET deleted key",
			Command:  "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n",
			Expected: "$-1\r\n",
		},
	}

	for _, test := range tests {
		fmt.Printf("Running test: %s\n", test.Name)
		_, err := conn.Write([]byte(test.Command))
		if err != nil {
			fmt.Printf("Write failed: %v\n", err)
			continue
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Read failed: %v\n", err)
			continue
		}
		resp := string(buffer[:n])

		if strings.TrimSpace(resp) == strings.TrimSpace(test.Expected) {
			fmt.Printf("✅ PASS: %s\n", test.Name)
		} else {
			fmt.Printf("❌ FAIL: %s\nExpected: %q\nGot     : %q\n", test.Name, test.Expected, resp)
		}
	}
}
