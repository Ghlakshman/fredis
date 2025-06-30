package main

import (
	"fredis/fredisdb"
	"fredis/server"
)

func main() {
	fredisStore := fredisdb.NewFredisStore("noeviction", 100)
	cmds := fredisdb.NewFredisCmds(fredisStore)
	s := server.NewServer(":6379", cmds)
	s.StartServer()
}
