package goflow

import (
	"sync"
)

type envSource[T any] struct {
	ch    <-chan envelope[T]
	once  sync.Once
	plain <-chan T
}
