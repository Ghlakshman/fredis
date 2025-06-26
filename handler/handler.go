package handler

import (
	"fmt"
	"fredis/fredisdb"
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

	switch strings.ToUpper(cmd[0]) {

	case "PING":
		if len(cmd) == 1 {
			return h.formatSimpleString("PONG"), nil
		} else if len(cmd) == 2 {
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
		return h.formatSimpleString("OK"), nil

	case "GET":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'GET'"), nil
		}
		key := cmd[1]
		val, err := h.Fcmds.GetValue(key)
		if err != nil || val == nil || val.Value == nil {
			return []byte("$-1\r\n"), nil
		}
		strVal, ok := val.Value.(string)
		if !ok {
			return h.formatError("ERR value is not a string"), nil
		}
		return h.formatBulkString(strVal), nil

	case "DEL":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'DEL'"), nil
		}
		if h.Fcmds.DelValue(cmd[1]) {
			return h.formatInteger(1), nil
		}
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
		return h.formatInteger(int(code)), nil

	case "TTL":
		if len(cmd) != 2 {
			return h.formatError("ERR wrong number of arguments for 'TTL'"), nil
		}
		ttl := h.Fcmds.TTL(cmd[1])
		return h.formatInteger(ttl), nil

	default:
		return h.formatError("ERR unknown command"), nil
	}
}
