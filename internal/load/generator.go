package load

import (
	"context"
	"sync"
	"time"
)

type HTTPStats struct {
	Latency    time.Duration
	Err        error
	StatusCode int
}

func RunHTTPTest(ctx context.Context, url string, connections int) []HTTPStats {
	var results []HTTPStats

	ch := make(chan []HTTPStats)
	var wg sync.WaitGroup

	for i := range connections {
		wg.Go(func() {
			MakeConnection(i, url, ch, ctx)
		})
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for workerResults := range ch {
		results = append(results, workerResults...)
	}

	return results
}
