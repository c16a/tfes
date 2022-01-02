package utils

import (
	"bufio"
	"encoding/json"
	"github.com/tfes-dev/tfes/pkg/schemas"
	"io"
)

func WriteToBufio(writer *bufio.Writer, msg *schemas.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = writer.Write(append(b, '\n'))
	if err != nil {
		return err
	}

	return writer.Flush()
}

func WriteToIo(writer io.Writer, msg *schemas.Message) error {
	w := bufio.NewWriter(writer)
	return WriteToBufio(w, msg)
}
