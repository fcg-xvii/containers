package containers

import "fmt"

// Структура стека
type Stack struct {
	list []interface{}
}

// Возвращает объект, находящийся на вершине стека
func (s *Stack) Peek() interface{} {
	if len(s.list) > 0 {
		return s.list[len(s.list)-1]
	}
	return nil
}

// Помещает объект на вершину стека
func (s *Stack) Push(vals ...interface{}) {
	s.list = append(s.list, vals...)
}

// Забирает объект с вершины стека и возвращает его
func (s *Stack) Pop() (res interface{}) {
	if len(s.list) > 0 {
		index := len(s.list) - 1
		res = s.list[index]
		s.list = s.list[:index]
	}
	return
}

// Возвращает размер стека
func (s *Stack) Len() int {
	return len(s.list)
}

// Возвращает вместимость структуры стека
func (s *Stack) Cap() int {
	return cap(s.list)
}

// Возвращает строку для предсавления элементов стека
func (s *Stack) String() string {
	res := fmt.Sprintf("Stack (len %v, cap %v)\n=====\n", len(s.list), cap(s.list))
	for i := len(s.list) - 1; i >= 0; i-- {
		res += fmt.Sprintf("%v: %v\n", i, s.list[i])
	}
	res += "=====\n"
	return res
}

// Забирает весь стек и возвращает его элементы в срезе
func (s *Stack) PopAll() []interface{} {
	res := make([]interface{}, len(s.list))
	copy(res, s.list)
	s.list = s.list[:0]
	return res
}

// Забирает стек до индекса (если в стеке, например, 5 элементов, индекс = 2, из стека будет взято 3 элемента)
func (s *Stack) PopAllIndex(index int) []interface{} {
	res := make([]interface{}, len(s.list)-index)
	copy(res, s.list[index:])
	s.list = s.list[:index]
	return res
}

// Разворачивает стек, забирает все элементы и возвращает их в срезе
func (s *Stack) PopAllReverse() []interface{} {
	res := make([]interface{}, len(s.list))
	for i := 0; i < len(s.list); i++ {
		res[i] = s.list[len(s.list)-i-1]
	}
	s.list = s.list[:0]
	return res
}

// Забирает элементы из стека до индекса, разворачивает результат и возвращает его в срезе
func (s *Stack) PopAllReverseIndex(index int) []interface{} {
	res := make([]interface{}, len(s.list)-index)
	for i := 0; i < len(s.list)-index; i++ {
		res[i] = s.list[len(s.list)-i-1]
	}
	s.list = s.list[:index]
	return res
}
