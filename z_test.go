package containers

import (
	"testing"
	"time"
)

type cacheStruct struct {
	id   int
	name string
}

var (
	cacher = NewCache(time.Second*10, time.Second*30)
)

func TestStack(t *testing.T) {
	t.Log("Test Stack")
	stack := new(Stack)

	t.Log("Test single append...")
	stack.Push(1)
	t.Log(stack)
	t.Log("Test items append...")
	stack.Push(2, 3, 4)
	t.Log(stack)
	t.Log("Test item pop")
	item := stack.Pop()
	t.Log("Item: ", item)
	t.Log(stack)
	t.Log("Test single append...")
	stack.Push(1)
	t.Log(stack)
	t.Log("Test pop all items")
	for stack.Len() > 0 {
		t.Log(stack.Pop())
		t.Log(stack)
	}
	t.Log("Test pop nil item")
	t.Log(stack.Pop())
	t.Log("Test pop all")
	stack.Push(0, 1, 2, 3, 4, 5)
	t.Log(stack.PopAll())
	t.Log(stack)
	stack.Push(0, 1, 2, 3, 4, 5)
	t.Log(stack.PopAllReverse())
	t.Log(stack)
	stack.Push(0, 1, 2, 3, 4, 5)
	t.Log(stack.PopAllReverseIndex(2))
	t.Log(stack)
	t.Log("Peek test")
	t.Log(stack.Peek())
	t.Log("Peek nil stack test")
	stack.PopAll()
	t.Log(stack.Peek())
}

func searchInCache(field interface{}) (res interface{}, check bool) {
	switch field.(type) {
	case int:
		id := field.(int)
		return cacher.Search(func(key, item interface{}) (vCheck bool) {
			return item.(*cacheStruct).id == id
		})
	case string:
		name := field.(string)
		return cacher.Search(func(key, item interface{}) (vCheck bool) {
			return item.(*cacheStruct).name == name
		})
	}
	return
}

func TestCache(t *testing.T) {
	t.Log("Set cache test")
	cacher.Set("te", &cacheStruct{10, "ten"})
	cacher.Set("le", &cacheStruct{11, "elleven"})
	t.Log(cacher.items)
	t.Log("Get cache test")
	t.Log(cacher.Get("ten"))
	t.Log("Search test")
	t.Log(searchInCache(10))
}
