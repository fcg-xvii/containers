package containers

import "fmt"

type Stack struct {
	list []interface{}
}

func (s *Stack) Push(vals ...interface{}) {
	s.list = append(s.list, vals...)
}

func (s *Stack) Pop() (res interface{}) {
	if len(s.list) > 0 {
		index := len(s.list) - 1
		res = s.list[index]
		s.list = s.list[:index]
	}
	return
}

func (s *Stack) Len() int {
	return len(s.list)
}

func (s *Stack) Cap() int {
	return cap(s.list)
}

func (s *Stack) String() string {
	res := fmt.Sprintf("Stack (len %v, cap %v)\n=====\n", len(s.list), cap(s.list))
	for i := len(s.list) - 1; i >= 0; i-- {
		res += fmt.Sprintf("%v: %v\n", i, s.list[i])
	}
	res += "=====\n"
	return res
}
