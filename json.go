package containers

import (
	"encoding/json"
	"io"
)

const (
	JSON_INVALID = iota
	JSON_ARRAY
	JSON_OBJECT
	JSON_VALUE
)

type JSONObject interface {
	DecodeJSON(*json.Decoder) error
}

func InitJSONDecoder(r io.Reader) *JSONDecoder {
	return &JSONDecoder{
		Decoder:  json.NewDecoder(r),
		embedded: NewStack(0),
	}
}

type JSONDecoder struct {
	*json.Decoder
	embedded     *Stack
	current      byte
	objectClosed bool
}

func (s *JSONDecoder) Token() (t json.Token, err error) {
	s.objectClosed = false
	if t, err = s.Decoder.Token(); err == nil {
		if delim, check := t.(json.Delim); check {
			switch delim {
			case '{':
				s.embedded.Push(JSON_OBJECT)
				current = JSON_OBJECT
			case '[':
				s.embedded.Push(JSON_ARRAY)
				current = JSON_ARRAY
			case '}', ']':
				s.embedded.Pop()
				s.objectClosed, s.current = true, JSON_INVALID
			}
		} else {
			current = JSON_VALUE
		}
	}
	return
}
