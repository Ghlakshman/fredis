package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

func ParseRESP(reader *bufio.Reader) (any, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 {
		return nil, errors.New("invalid RESP input")
	}

	prefix := line[0]
	data := string(line[1 : len(line)-2]) // remove \r\n

	switch prefix {
	case '*':
		count, err := strconv.Atoi(data)
		if err != nil {
			return nil, errors.New("array count must be numeric")
		}
		var items []any
		for i := 0; i < count; i++ {
			val, err := ParseRESP(reader)
			if err != nil {
				return nil, err
			}
			items = append(items, val)
		}
		return items, nil

	case '+', '-', ':':
		return data, nil

	case '$':
		strlen, err := strconv.Atoi(data)
		if err != nil {
			return nil, errors.New("invalid bulk string length")
		}
		if strlen == -1 {
			return nil, nil
		}
		buf := make([]byte, strlen)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}
		reader.Discard(2)
		return string(buf), nil

	default:
		return nil, fmt.Errorf("unknown RESP type: %c", prefix)
	}
}
