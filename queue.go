package LockFree

import (
	"sync/atomic"
	"unsafe"
)

type Queue[T any] struct {
	head *Node[T]
	tail *Node[T]
	size int
}

type Node[T any] struct {
	next  *Node[T]
	value T
}

func (q *Queue[T]) Push(v T) {
	ok := false
	node := &Node[T]{
		value: v,
		next:  nil,
	}
	for !ok {
		tail := (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail))))
		next := (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&tail.next))))
		// 比较队头是否被更新
		if tail == (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)))) {
			// 此时 队列没有被其他线程插入新值)
			if next == nil {
				// 比较队列是否仍然没有被更新, 未更新则将tail.next置为新值
				// (这应该是原子操作...嗯...一定是, 是不可中断的呢)
				ok = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&tail.next)),
					nil, unsafe.Pointer(node))
			}
			// 如果队列的tail没有被更新, 就将tail向后移动一个节点
			// 首先, 代码运行到这
			// 如果没有在当前线程成功更新:
			//	说明其他节点已经更新了, 那么更新队列tail节点状态是合理的,
			//  此时, 其他节点在其线程进行比较的时候不会触发swap, 所以这里更新没有影响
			// 如果在当前线程更新成功:
			//  那么队列的tail需要更新, 无论是当前线程还是其他线程, 都只会有一个成功更新队列
			//  所以还是没有影响.
			// (话说这段代码挺难理解的）
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)),
				unsafe.Pointer(tail), atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&tail.next))))
		}
	}
}

func (q *Queue[T]) Pop() T {
	ok := false
	for !ok {
		head := (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head))))
		tail := (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail))))
		next := (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&head.next))))
		if head == (*Node[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)))) {
			if head == tail { // 队列是空的 不能弹出
				panic("Empty Queue")
				return nil
			}
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), // 同样的思路, 没被更新就进行更新
				unsafe.Pointer(tail), unsafe.Pointer(next))
		} else {
			// 操作同上 性质是一样的 如果队头不匹配了 就试着更新它 不必等待更新它的线程更新
			ok = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)),
				unsafe.Pointer(head), unsafe.Pointer(next))
			if ok {
				return next.value
			}
		}
	}
}
