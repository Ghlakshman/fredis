package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Example: RESP for `PING`
	command := "*6\r\n+OK\r\n:123\r\n$5\r\nhello\r\n$-1\r\n*-1\r\n*3\r\n+foo\r\n:456\r\n$3\r\nbar\r\n"

	_, err = conn.Write([]byte(command))
	if err != nil {
		fmt.Println("Write failed:", err)
		return
	}

	// Optional: Read response (if your server replies)
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Read failed:", err)
		return
	}
	fmt.Println("Response:", string(buffer[:n]))
}
