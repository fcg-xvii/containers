package containers

import "testing"

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
