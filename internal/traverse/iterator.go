package traverse

import "grapher/internal/graph"

// Iterator 通用图遍历接口
type Iterator[T comparable] interface {
	// HasNext 返回一个布尔值，指示是否有更多元素可供迭代。如果有更多元素，则返回 true。否则，返回 false。
	HasNext() bool

	// Next 返回序列中下一个要迭代的元素。如果没有更多元素，则返回 nil。它还会将迭代器推进到下一个元素。
	Next() *graph.Node[T]

	// Iterate 遍历序列中的所有元素，并对每个元素调用提供的回调函数。
	Iterate(func(*graph.Node[T]) error) error
}

// Direction 遍历方向枚举
type Direction int

const (
	Outgoing Direction = iota // 向下遍历 (默认)
	Incoming                  // 向上遍历
)
