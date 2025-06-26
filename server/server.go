package server

import (
	"bufio"
	"fmt"
	"fredis/fredisdb"
	"fredis/handler"
	"fredis/resp"
	"io"
	"log"
	"net"
	"time"
)

type Server struct {
	serverAddr string
	client     net.Listener
	fdCmds     *fredisdb.FredisCmds
}

func NewServer(serverAddr string, fdCmds *fredisdb.FredisCmds) *Server {
	return &Server{
		serverAddr: serverAddr,
		fdCmds:     fdCmds,
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
		handler := handler.Handler{
			Conn:  conn,
			Fcmds: serve.fdCmds,
		}
		go serve.readConnections(conn, &handler)
	}
}

func (serve *Server) readConnections(conn net.Conn, hndlr *handler.Handler) {
	defer func() {
		log.Printf("Client disconnected: %s\n", conn.RemoteAddr())
		conn.Close()
	}()

	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))

	reader := bufio.NewReader(conn)
	for {
		// line, err := reader.ReadBytes('\n')
		line, err := resp.ParseRESP(reader)
		if err != nil {
			if err == io.EOF {
				log.Printf("Client %s closed the connection\n", conn.RemoteAddr())
				return
			}
			// log.Println("Error in reading bytes from connection", conn.RemoteAddr(), err)
			return
		}
		respOutBytes, _ := hndlr.HandleCommand(line)
		conn.Write(respOutBytes)
	}
}
