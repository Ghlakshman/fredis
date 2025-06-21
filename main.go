package main

import "fredis/server"

func main() {
	s := server.NewServer(":6379")
	s.StartServer()
}
