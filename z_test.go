package containers

import (
	"log"
	"testing"
	"time"
)

type cacheStruct struct {
	id   int
	name string
}

func init() {
	cacher.clearPrepare = func(items []interface{}) {
		log.Println(items, cacher.items)
	}
}

var (
	cacher       = NewCache(time.Second*10, time.Second*15, nil)
	listCallback = func(src []byte, store func(interface{})) error {
		for i := 0; i < 10; i++ {
			store(i)
		}
		return nil
	}
	mapCallback = func(src []byte, store func(interface{}, interface{})) error {
		for i := 0; i < 100000; i++ {
			store(i, i+1)
		}
		return nil
	}
	fileList = NewFileList("z-content", listCallback)
	fileMap  = NewFileMap("z-content", mapCallback)
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

func TestCache(t *testing.T) {
	t.Log("Set cache test")
	cacher.Set("te", &cacheStruct{10, "ten"})
	cacher.Set("le", &cacheStruct{11, "elleven"})
	t.Log(cacher.items)
	t.Log("Get cache test")
	t.Log(cacher.Get("ten", nil))
	t.Log("Search test")

	for i := 0; i < 500; i++ {
		go func() {
			cacher.Get("te", nil)
		}()
	}

	time.Sleep(time.Second * 600)
}

func TestFile(t *testing.T) {
	//t.Log(list.Len())
	//t.Log(f.Len())

	for i := 0; i < 500; i++ {
		go func() {
			fileList.Get(1)
		}()
	}
	time.Sleep(time.Second)
	t.Log(fileMap.Len())
}

func BenchmarkListFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fileList.Len()
	}
}

func BenchmarkMapFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fileMap.Len()
	}
}

func BenchmarkMapFileRead(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fileMap.Get(500)
	}
}
