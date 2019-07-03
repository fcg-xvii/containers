package containers

import (
	_ "fmt"
	"log"
	"os"
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
	val, check := cacher.GetOrCreate(10, func(val interface{}) bool {
		log.Println("CHECK")
		return true
	}, func(key interface{}) (rKey interface{}, rVal interface{}, rCheck bool) {
		return 10, "ten", true
	})

	log.Println("1)", val, check, cacher.Keys())

	val, check = cacher.GetOrCreate(10, func(val interface{}) bool {
		log.Println("CHECK")
		return true
	}, func(key interface{}) (rKey interface{}, rVal interface{}, rCheck bool) {
		return 10, "ten", true
	})

	log.Println("2)", val, check)
	cacher.Set("ok", nil)
	log.Println(cacher.Get("ok", nil))
}

func TestFile(t *testing.T) {
	for i := 0; i < 500; i++ {
		go func() {
			fileList.Get(1)
		}()
	}
	time.Sleep(time.Second)
	t.Log(fileMap.Len())
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

type TObject struct {
	id          int
	name, value string
	armed       []int
	child       *TObject
}

// Raw dicode
/*func (s *TObject) DecodeJSON(dec *JSONDecoder) error {
	_, err := dec.Token()
	if err != nil {
		return err
	}
	if dec.Current() != JSON_OBJECT {
		return fmt.Errorf("Expected object, not %v", dec.Current())
	}
	el := dec.EmbeddedLevel()
	for el <= dec.EmbeddedLevel() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		if dec.Current() == JSON_VALUE && dec.IsObjectKey() {
			var ptr interface{}
			switch t.(string) {
			case "id":
				ptr = &s.id
			case "name":
				ptr = &s.name
			case "value":
				ptr = &s.value
			case "armed":
				ptr = &s.armed
			case "child":
				s.child = new(TObject)
				ptr = s.child
			}
			if ptr == nil {
				if err = dec.Next(); err != nil {
					return err
				}
			} else {
				if err = dec.Decode(ptr); err != nil {
					return err
				}
			}
		}
	}
	return nil
}*/

// Object decode
func (s *TObject) DecodeJSON(dec *JSONDecoder) error {
	return dec.DecodeObject(func(field string) (ptr interface{}, err error) {
		switch field {
		case "id":
			ptr = &s.id
		case "name":
			ptr = &s.name
		case "value":
			ptr = &s.value
		case "armed":
			ptr = &s.armed
		case "child":
			s.child = new(TObject)
			ptr = s.child
		}
		return
	})
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func TestJSON(t *testing.T) {
	var obj TObject
	f, err := os.Open("z-json-content.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := InitJSONDecoder(f)
	if err := obj.DecodeJSON(dec); err != nil {
		t.Fatal(err)
	}
	log.Println(obj)
	log.Println(*obj.child)
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
