package handler

import (
	"fmt"
	"fredis/fredisdb"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	Conn  net.Conn
	Fcmds *fredisdb.FredisCmds
}

// RESP formatters
func (h *Handler) formatBulkString(s string) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
}

func (h *Handler) formatError(msg string) []byte {
	log.Printf("[ERROR] %s", msg)
	return []byte(fmt.Sprintf("-%s\r\n", msg))
}

func (h *Handler) formatSimpleString(s string) []byte {
	return []byte(fmt.Sprintf("+%s\r\n", s))
}

func (h *Handler) formatInteger(n int) []byte {
	return []byte(fmt.Sprintf(":%d\r\n", n))
}

// Main command handler
func (h *Handler) HandleCommand(cmd []string) ([]byte, error) {
	if len(cmd) == 0 {
		return h.formatError("empty command"), nil
	}

	log.Printf("[COMMAND] Received: %v", cmd)

	switch strings.ToUpper(cmd[0]) {

	case "PING":
		if len(cmd) == 1 {
			log.Println("[PING] No message")
			return h.formatSimpleString("PONG"), nil
		} else if len(cmd) == 2 {
			log.Printf("[PING] Echo message: %s", cmd[1])
			return h.formatBulkString(cmd[1]), nil
		}
		return h.formatError("ERR wrong number of arguments for 'PING'"), nil

	case "SET":
		if len(cmd) < 3 {
			return h.formatError("ERR wrong number of arguments for 'SET'"), nil
		}
		key := cmd[1]
		val := &fredisdb.Value{
			Value:      cmd[2],
			LastAccess: time.Now(),
			IsVolatile: false,
			Expiry:     nil,
		}
		h.Fcmds.SetValue(key, val)
		log.Printf("[SET] Key: %s, Value: %s", key, cmd[2])
		return h.formatSimpleString("OK"), nil

	case "GET":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'GET'"), nil
		}
		key := cmd[1]
		val, err := h.Fcmds.GetValue(key)
		if err != nil || val == nil || val.Value == nil {
			log.Printf("[GET] Key not found or expired: %s", key)
			return []byte("$-1\r\n"), nil
		}
		strVal, ok := val.Value.(string)
		if !ok {
			log.Printf("[GET] Key %s has non-string value", key)
			return h.formatError("ERR value is not a string"), nil
		}
		log.Printf("[GET] Key: %s, Value: %s", key, strVal)
		return h.formatBulkString(strVal), nil

	case "DEL":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'DEL'"), nil
		}
		key := cmd[1]
		if h.Fcmds.DelValue(key) {
			log.Printf("[DEL] Key deleted: %s", key)
			return h.formatInteger(1), nil
		}
		log.Printf("[DEL] Key not found: %s", key)
		return h.formatInteger(0), nil

	case "EXPIRE":
		if len(cmd) != 3 {
			return h.formatError("ERR wrong number of arguments for 'EXPIRE'"), nil
		}
		seconds, err := strconv.ParseInt(cmd[2], 10, 64)
		if err != nil {
			return h.formatError("ERR value is not an integer or out of range"), nil
		}
		code, _ := h.Fcmds.SetExpiry(cmd[1], seconds)
		log.Printf("[EXPIRE] Key: %s, Seconds: %d, Result Code: %d", cmd[1], seconds, code)
		return h.formatInteger(int(code)), nil

	case "TTL":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'TTL'"), nil
		}
		ttl := h.Fcmds.TTL(cmd[1])
		log.Printf("[TTL] Key: %s, TTL: %d", cmd[1], ttl)
		return h.formatInteger(ttl), nil

	case "CONFIG":
		if len(cmd) != 4 || strings.ToUpper(cmd[1]) != "SET" || strings.ToLower(cmd[2]) != "eviction-policy" {
			return h.formatError("ERR usage: CONFIG SET eviction-policy <policy>"), nil
		}

		policy := strings.ToLower(cmd[3])
		if !fredisdb.IsValidEvictionPolicy(policy) {
			return h.formatError("ERR invalid eviction policy"), nil
		}
		h.Fcmds.FredisDb.EvictionPolicy = fredisdb.EvictionPolicy(policy)
		log.Printf("[CONFIG] Eviction policy set to: %s", policy)
		return h.formatSimpleString("OK"), nil

	default:
		log.Printf("[UNKNOWN] Unknown command: %s", cmd[0])
		return h.formatError("ERR unknown command"), nil
	}
}
