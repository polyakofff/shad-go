//go:build !solution

package treeiter

type Node[T any] interface {
	Left() T
	Right() T
	comparable
}

func DoInOrder[T Node[T]](node T, f func(T)) {
	var zero T
	if node == zero {
		return
	}

	DoInOrder(node.Left(), f)
	f(node)
	DoInOrder(node.Right(), f)
}
