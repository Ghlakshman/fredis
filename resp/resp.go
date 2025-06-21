package resp

import (
	"bufio"
	"fmt"
)

func RespParser(reader *bufio.Reader) (any, error) {
	fmt.Println(reader.ReadBytes('\n'))
	return nil, nil
}
