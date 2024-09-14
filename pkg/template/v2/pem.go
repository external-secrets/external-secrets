//Copyright External Secrets Inc. All Rights Reserved

package template

import (
	"bytes"
	"encoding/pem"
	"errors"
	"strings"
)

const (
	errJunk = "error filtering pem: found junk"
)

func filterPEM(pemType, input string) (string, error) {
	data := []byte(input)
	var blocks []byte
	var block *pem.Block
	var rest []byte
	for {
		block, rest = pem.Decode(data)
		data = rest

		if block == nil {
			break
		}
		if !strings.EqualFold(block.Type, pemType) {
			continue
		}

		var buf bytes.Buffer
		err := pem.Encode(&buf, block)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, buf.Bytes()...)
	}

	if len(blocks) == 0 && len(rest) != 0 {
		return "", errors.New(errJunk)
	}

	return string(blocks), nil
}

func pemEncode(thing, kind string) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: kind, Bytes: []byte(thing)})
	return buf.String(), err
}
