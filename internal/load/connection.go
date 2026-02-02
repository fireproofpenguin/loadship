package load

import (
	"context"
	"io"
	"net/http"
	"time"
)

func MakeConnection(id int, url string, channel chan []HTTPStats, ctx context.Context) {
	defaultTimeout := 30 * time.Second

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	var results []HTTPStats

	for {
		if ctx.Err() != nil {
			channel <- results
			return
		}

		reqStart := time.Now()
		resp, err := client.Get(url)

		if err != nil {
			results = append(results, HTTPStats{Timestamp: reqStart, Err: err})
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		latency := time.Since(reqStart)

		results = append(results, HTTPStats{
			Timestamp:  reqStart,
			Latency:    latency,
			StatusCode: resp.StatusCode,
		})
	}
}
