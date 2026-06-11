package sets

import "sync"

type SyncedSet[T comparable] struct {
	set Set[T]
	mu  sync.Mutex
}

func NewSyncedSet[T comparable]() *SyncedSet[T] {
	return &SyncedSet[T]{
		set: make(Set[T]),
	}
}
