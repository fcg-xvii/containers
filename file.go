package containers

import (
	"errors"
	"os"
	"sync"
)

var (
	errOutOfRange = errors.New("Index out of range")
)

type File struct {
	path     string
	modified int64
	locker   *sync.RWMutex
}

func (s *File) checkModified() (bool, error) {
	if info, err := os.Stat(s.path); err != nil {
		return false, err
	} else {
		return info.ModTime().UnixNano() == s.modified, nil
	}
}

type FileList struct {
	*File
	items []interface{}
}

func (s *FileList) Get(index int) (interface{}, error) {
	if index < 0 || index >= len(s.items) {
		return nil, errOutOfRange
	}
	s.locker.RLock()
	if check, err := s.checkModified() err != nil {
		s.locker.RUnlock()
		return nil, err
	} else {
		
	}
}
