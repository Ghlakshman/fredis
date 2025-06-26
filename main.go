package main

import (
	"fredis/fredisdb"
	"fredis/server"
)

func main() {
	fredisStore := fredisdb.NewFredisStore("", 100000)
	cmds := fredisdb.NewFredisCmds(fredisStore)
	s := server.NewServer(":6379", cmds)
	s.StartServer()
}
