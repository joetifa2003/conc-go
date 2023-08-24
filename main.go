package main

import (
	"context"
	"sync"
)

type Resault[T any] struct {
	value T
	err   error
}

func (r *Resault[T]) Get() (T, error) {
	return r.value, r.err
}

func NewResaultSome[T any](value T) Resault[T] {
	return Resault[T]{value: value}
}

func NewResaultErr[T any](err error) Resault[T] {
	return Resault[T]{err: err}
}

func Stream[T any](ctx context.Context, items []T) <-chan T {
	ch := make(chan T)

	go func() {
		defer close(ch)
		for _, item := range items {
			select {
			case <-ctx.Done():
				return

			case ch <- item:
			}
		}
	}()

	return ch
}

func Chunk[I any](ctx context.Context, items <-chan I, chunkSize int) <-chan []I {
	ch := make(chan []I)

	go func() {
		defer close(ch)
		chunk := make([]I, 0, chunkSize)

		for item := range items {
			chunk = append(chunk, item)
			if len(chunk) == chunkSize {
				select {
				case <-ctx.Done():
					return
				case ch <- chunk:
					chunk = make([]I, 0, chunkSize)
				}
			}
		}

		if len(chunk) != 0 {
			select {
			case <-ctx.Done():
				return
			case ch <- chunk:
			}
		}
	}()

	return ch
}

func ProcessItems[I any, O any](
	ctx context.Context,
	inputCh <-chan I,
	workers int,
	process func(I, chan<- O) error,
) <-chan Resault[O] {
	resCh := make(chan Resault[O])
	oCh := make(chan O)
	wg := sync.WaitGroup{}
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inputCh:
					if !ok {
						return
					}
					err := process(item, oCh)
					if err != nil {
						resCh <- NewResaultErr[O](err)
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(oCh)
	}()

	go func() {
		defer close(resCh)
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-oCh:
				if !ok {
					return
				}
				resCh <- NewResaultSome[O](item)
			}
		}
	}()

	return resCh
}

func main() {
	ctx := context.Background()
	inputCh := make(chan int)
	go func(chan<- int) {
		defer close(inputCh)
		for i := 2; i < 1000; i++ {
			inputCh <- i
		}
	}(inputCh)
	inputChChunked := Chunk(ctx, inputCh, 1)

	resCh := ProcessItems(ctx, inputChChunked, 1, func(i []int, res chan<- int) error {
		for _, item := range i {
			if checkPrime(item) {
				res <- item
			}
		}

		return nil
	})

	for res := range resCh {
		println(res.value)
	}
}

func checkPrime(n int) bool {
	for i := 2; i < (n/2)+1; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}
