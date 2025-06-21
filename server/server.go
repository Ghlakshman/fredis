package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Server struct {
	serverAddr string
	client     net.Listener
}

func NewServer(serverAddr string) *Server {
	return &Server{
		serverAddr: serverAddr,
	}
}

func (serve *Server) StartServer() error {
	fmt.Printf("Starting Server at localhost%s", serve.serverAddr)
	ln, err := net.Listen("tcp", serve.serverAddr)
	if err != nil {
		return err
	}
	serve.client = ln
	serve.acceptConnections()
	return nil
}

func (serve *Server) acceptConnections() {

	for {
		conn, err := serve.client.Accept()
		log.Println("New Clinet Connected", conn.RemoteAddr())
		if err != nil {
			log.Println("Error in Accepting Connection", conn.RemoteAddr(), err)
			continue
		}
		go serve.readConnections(conn)
	}
}

func (serve *Server) readConnections(conn net.Conn) {
	defer func() {
		log.Printf("Client disconnected: %s\n", conn.RemoteAddr())
		conn.Close()
	}()

	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Client %s closed the connection\n", conn.RemoteAddr())
				return
			}
			log.Println("Error in reading bytes from connection", conn.RemoteAddr(), err)
			return
		}
		fmt.Printf("Bytes Read from %s are %s", conn.RemoteAddr(), string(line))
		log.Printf("Bytes Read from %s are %s", conn.RemoteAddr(), string(line))
	}
}
