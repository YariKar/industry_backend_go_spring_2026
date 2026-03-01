package main

import (
	"context"
	"errors"
	"sync"
)

func ParallelMap[T any, R any](
	parentCtx context.Context,
	workers int,
	in []T,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	if workers <= 0 {
		return nil, errors.New("workers must be positive")
	}

	if len(in) == 0 {
		return []R{}, nil
	}

	if err := parentCtx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	type job struct {
		idx int
		val T
	}
	jobs := make(chan job, len(in))

	type result struct {
		idx int
		res R
		err error
	}
	results := make(chan result, len(in))

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					r, err := fn(ctx, job.val)
					results <- result{idx: job.idx, res: r, err: err}
				}
			}
		}()
	}

	for i, v := range in {
		select {
		case <-ctx.Done():
			close(jobs)
			goto waitAndCollect
		default:
			jobs <- job{idx: i, val: v}
		}
	}
	close(jobs)

waitAndCollect:
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]R, len(in))
	var firstErr error
	var once sync.Once

	for res := range results {
		if res.err != nil {
			once.Do(func() {
				firstErr = res.err
				cancel()
			})
		}
		out[res.idx] = res.res
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}
