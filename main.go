package main

import (
	"bufio"
	"fredis/fredisdb"
	"fredis/handler"
	"fredis/resp"
	"fredis/server"
	"io"
	"log"
	"os"

	"github.com/pelletier/go-toml"
)

type FredisConfig struct {
	EvictionPolicy string `toml:"eviction_policy"`
	MaxEntries     uint64 `toml:"max_entries"`
}

func LoadConfig(path string) (*FredisConfig, error) {
	config := &FredisConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = toml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ReplayAOF replays commands from an AOF file using a stateless handler.
func ReplayAOF(filepath string, h *handler.Handler) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		cmds, err := resp.ParseRESP(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error parsing AOF command: %v", err)
			continue
		}
		_, err = h.HandleCommand(cmds)
		if err != nil {
			log.Printf("Error executing AOF command: %v", err)
		}
	}
	log.Println("[AOF] Restore complete.")
	return nil
}

func main() {
	var fredisStore *fredisdb.FredisStore

	fredisConfig, err := LoadConfig("config.toml")
	if err != nil {
		log.Println("[Main] Error Loading Config from the toml file!! Starting Server with Default Config: ['noeviction','1000 Entries']")
		fredisStore = fredisdb.NewFredisStore("noeviction", 100)
	} else {
		log.Printf("Loading Config from the toml file!! ['Eviction Policy':%s,MaxEntries:%d]", fredisConfig.EvictionPolicy, fredisConfig.MaxEntries)
		fredisStore = fredisdb.NewFredisStore(fredisdb.EvictionPolicy(fredisConfig.EvictionPolicy), fredisConfig.MaxEntries)
	}

	aof := fredisdb.NewAOF("aof/fredis.aof")
	cmds := fredisdb.NewFredisCmds(fredisStore, aof)
	cmds.IsReplaying = true
	aofHandler := &handler.Handler{
		Conn:  nil,
		Fcmds: cmds,
	}
	AOFRrr := ReplayAOF("aof/fredis.aof", aofHandler)
	if AOFRrr != nil {
		log.Printf("[WARN] AOF Replay failed: %v", AOFRrr)
	}
	cmds.IsReplaying = false
	s := server.NewServer(":6379", cmds)
	s.StartServer()
}
