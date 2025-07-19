// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package syncutil

import (
	"runtime"
	"slices"
	"sync"
)

func Map[S ~[]E, E, R any](items S, limit int, out []R, fn func(subitems S) (R, error)) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	if limit <= 0 {
		limit = runtime.GOMAXPROCS(0)
	}

	if limit > len(items) {
		limit = len(items)
	}

	out = slices.Grow(out, limit)[:len(out)+limit]
	err := Distribute(items, limit, func(group int, step int, subitems S) error {
		res, err := fn(subitems)
		out[group] = res
		return err
	})

	return out, err
}

func Distribute[S ~[]E, E any](items S, limit int, fn func(group int, step int, subitems S) error) error {
	if len(items) == 0 {
		return nil
	}

	if limit <= 0 {
		limit = runtime.GOMAXPROCS(0)
	}

	if limit > len(items) {
		limit = len(items)
	}

	step := len(items) / limit
	var muGerr sync.Mutex
	var gerr error
	var wg sync.WaitGroup
	wg.Add(limit)
	for g := range limit {
		go func() {
			defer wg.Done()
			var subset S
			if g < limit-1 {
				subset = items[g*step : (g+1)*step]
			} else {
				subset = items[g*step:]
			}
			if err := fn(g, step, subset); err != nil {
				muGerr.Lock()
				if gerr == nil {
					gerr = err
				}
				muGerr.Unlock()
			}
		}()
	}
	wg.Wait()
	return gerr
}

type Pool[T any] struct {
	pool sync.Pool
}

func NewPool[T any](fn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return fn()
			},
		},
	}
}

func (p *Pool[T]) Put(v T) {
	p.pool.Put(v)
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func TryRecv[T any](ch <-chan T) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
