package main

import (
	"context"
	"fmt"
	"testing"
)

const N = 10000

func BenchmarkPrimeConc(b *testing.B) {
	benchs := [][]int{
		{1, 1}, {1, 2}, {1, 4}, {1, 8},
		{5, 1}, {5, 2}, {5, 4}, {5, 8},
		{15, 1}, {15, 2}, {15, 4}, {15, 8},
		{30, 1}, {30, 2}, {30, 4}, {30, 8},
		{60, 1}, {60, 2}, {60, 4}, {60, 8},
	}

	for _, benchCase := range benchs {
		chunkSize := benchCase[0]
		workers := benchCase[1]
		b.Run(fmt.Sprintf("With chunk size %d and workers %d", chunkSize, workers), func(b *testing.B) {
			bench(b, chunkSize, workers)
		})
	}
}

func bench(b *testing.B, chunkSize int, workers int) {
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		inputCh := make(chan int)
		go func(chan<- int) {
			defer close(inputCh)
			for i := 2; i < N; i++ {
				inputCh <- i
			}
		}(inputCh)
		inputChChunked := Chunk(ctx, inputCh, chunkSize)

		resCh := ProcessItems(ctx, inputChChunked, workers, func(i []int, res chan<- int) error {
			for _, item := range i {
				if checkPrime(item) {
					res <- item
				}
			}

			return nil
		})

		for range resCh {
		}
	}
}
