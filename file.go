package containers

import (
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
)

type FileIndexCallback func([]byte, func(interface{})) error
type FileMapCallback func([]byte, func(interface{}, interface{})) error

type file struct {
	path                string
	modified            int64
	locker              *sync.RWMutex
	parseMethod         func([]byte) error
	updatePrepareMethod func()
}

func (s *file) update() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if atomic.LoadInt64(&s.modified) != info.ModTime().UnixNano() {
		s.locker.Lock()
		if s.modified == info.ModTime().UnixNano() {
			s.locker.Unlock()
			return nil
		}
		if s.updatePrepareMethod != nil {
			s.updatePrepareMethod()
		}
		var src []byte
		if src, err = ioutil.ReadFile(s.path); err == nil {
			if err = s.parseMethod(src); err == nil {
				atomic.StoreInt64(&s.modified, info.ModTime().UnixNano())
			}
		}
		s.locker.Unlock()
		return err
	}
	return nil
}

func (s *file) ModifiedTimestamp() int64 {
	return atomic.LoadInt64(&s.modified)
}

////////////////////////////////////////////////////////////////////////////

func NewFileObject(path string, parseCallback FileIndexCallback) *FileObject {
	f := &FileObject{parseCallback: parseCallback}
	f.file = &file{path: path, locker: new(sync.RWMutex), parseMethod: f.parse, updatePrepareMethod: nil}
	return f
}

type FileObject struct {
	*file
	obj           atomic.Value
	parseCallback FileIndexCallback
}

func (s *FileObject) set(val interface{}) {
	s.obj.Store(val)
}

func (s *FileObject) Get() (interface{}, error) {
	if err := s.update(); err != nil {
		return nil, err
	}
	return s.obj.Load(), nil
}

func (s *FileObject) parse(src []byte) error {
	return s.parseCallback(src, s.set)
}

////////////////////////////////////////////////////////////////////////////

func NewFileList(path string, parseCallback FileIndexCallback) *FileList {
	f := &FileList{parseCallback: parseCallback}
	f.file = &file{path: path, locker: new(sync.RWMutex), parseMethod: f.parse, updatePrepareMethod: f.updatePrepare}
	return f
}

type FileList struct {
	*file
	items         []interface{}
	parseCallback FileIndexCallback
}

func (s *FileList) updatePrepare() {
	s.items = nil
}

func (s *FileList) append(val interface{}) {
	s.items = append(s.items, val)
}

func (s *FileList) parse(src []byte) error {
	return s.parseCallback(src, s.append)
}

func (s *FileList) Get(index int) (interface{}, error) {
	if err := s.update(); err != nil {
		return nil, err
	}
	s.locker.RLock()
	res := s.items[index]
	s.locker.RUnlock()
	return res, nil
}

func (s *FileList) Len() (int, error) {
	if err := s.update(); err != nil {
		return 0, err
	}
	s.locker.RLock()
	l := len(s.items)
	s.locker.RUnlock()
	return l, nil
}

func (s *FileList) Range(callback func(int, interface{}) bool) {
	if err := s.update(); err != nil {
		return
	}
	s.locker.RLock()
	for i, v := range s.items {
		if !callback(i, v) {
			s.locker.RUnlock()
			return
		}
	}
	s.locker.RUnlock()
}

////////////////////////////////////////////////////////////////////////////

func NewFileMap(path string, parseCallback FileMapCallback) *FileMap {
	f := &FileMap{parseCallback: parseCallback, items: make(map[interface{}]interface{})}
	f.file = &file{path: path, locker: new(sync.RWMutex), parseMethod: f.parse, updatePrepareMethod: f.updatePrepare}
	return f
}

type FileMap struct {
	*file
	items         map[interface{}]interface{}
	parseCallback FileMapCallback
}

func (s *FileMap) updatePrepare() {
	s.items = make(map[interface{}]interface{})
}

func (s *FileMap) append(key, val interface{}) {
	s.items[key] = val
}

func (s *FileMap) parse(src []byte) error {
	s.parseCallback(src, s.append)
	return nil
}

func (s *FileMap) Get(key interface{}) (interface{}, bool, error) {
	if err := s.update(); err != nil {
		return nil, false, err
	}
	s.locker.RLock()
	res, check := s.items[key]
	s.locker.RUnlock()
	return res, check, nil
}

func (s *FileMap) Len() (int, error) {
	if err := s.update(); err != nil {
		return 0, err
	}
	s.locker.RLock()
	l := len(s.items)
	s.locker.RUnlock()
	return l, nil
}

func (s *FileMap) Range(callback func(interface{}, interface{}) bool) {
	if err := s.update(); err != nil {
		return
	}
	s.locker.RLock()
	for k, v := range s.items {
		if !callback(k, v) {
			s.locker.RUnlock()
			return
		}
	}
	s.locker.RUnlock()
}
