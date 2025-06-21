package main

import "github.com/Ghlakshman/fredis/server"

func main() {
	s := server.NewServer(":6379")
	s.StartServer()
}
