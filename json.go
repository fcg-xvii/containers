package containers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type JSONTokenType byte

const (
	JSON_INVALID JSONTokenType = iota
	JSON_ARRAY
	JSON_OBJECT
	JSON_VALUE
)

func (s JSONTokenType) String() string {
	switch s {
	case JSON_INVALID:
		return "JSON_INVALID"
	case JSON_ARRAY:
		return "JSON_ARRAY"
	case JSON_OBJECT:
		return "JSON_OBJECT"
	case JSON_VALUE:
		return "JSON_VALUE"
	default:
		return "JSON_UNDEFINED"
	}
}

type JSONObject interface {
	DecodeJSON(*JSONDecoder) error
}

func InitJSONDecoderFromSource(src []byte) *JSONDecoder {
	r := bytes.NewReader(src)
	return InitJSONDecoder(r)
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
	current      JSONTokenType
	objectkey    bool
	objectClosed bool
}

func (s *JSONDecoder) IsObjectKey() bool      { return s.objectkey }
func (s *JSONDecoder) IsObjectClosed() bool   { return s.objectClosed }
func (s *JSONDecoder) Current() JSONTokenType { return s.current }
func (s *JSONDecoder) EmbeddedLevel() int     { return s.embedded.Len() }

func (s *JSONDecoder) Token() (t json.Token, err error) {
	s.objectClosed = false
	if t, err = s.Decoder.Token(); err == nil {
		if delim, check := t.(json.Delim); check {
			s.objectkey = false
			switch delim {
			case '{':
				s.embedded.Push(JSON_OBJECT)
				s.current = JSON_OBJECT
			case '[':
				s.embedded.Push(JSON_ARRAY)
				s.current = JSON_ARRAY
			case '}', ']':
				s.embedded.Pop()
				s.objectClosed, s.current = true, JSON_INVALID
				if s.embedded.Len() > 0 {
					s.current = s.embedded.Peek().(JSONTokenType)
				}
			}
		} else {
			if s.current == JSON_OBJECT {
				s.objectkey = !s.objectkey
			}
			s.current = JSON_VALUE
		}
	}
	return
}

func (s *JSONDecoder) Next() error {
	if _, err := s.Token(); err != nil {
		return err
	}
	switch s.current {
	case JSON_ARRAY, JSON_OBJECT:
		{
			stackLen := s.embedded.Len()
			for s.embedded.Len() >= stackLen {
				if _, err := s.Token(); err != nil {
					return err
				}
			}
			return nil
		}
	default:
		return nil
	}
}

func (s *JSONDecoder) Decode(v interface{}) error {
	if jsonObj, check := v.(JSONObject); check {
		return jsonObj.DecodeJSON(s)
	} else {
		return s.Decoder.Decode(v)
	}
}

func (s *JSONDecoder) DecodeObject(fieldRequest func(string) (interface{}, error)) error {
	if _, err := s.Token(); err != nil {
		return err
	}
	if s.current != JSON_OBJECT {
		return fmt.Errorf("Expected object, not %v", s.current)
	}
	el := s.EmbeddedLevel()
	for el <= s.EmbeddedLevel() {
		t, err := s.Token()
		if err != nil {
			return err
		}
		if s.Current() == JSON_VALUE && s.IsObjectKey() {
			if ptr, err := fieldRequest(t.(string)); err != nil {
				return err
			} else if ptr != nil {
				if err = s.Decode(ptr); err != nil {
					return err
				}
			} else {
				if err = s.Next(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
