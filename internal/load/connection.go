package load

import (
	"io"
	"net/http"
	"time"
)

func RequestLoop(id int, url string, startTime time.Time, duration time.Duration, channel chan []requestResult) {
	defaultTimeout := 30 * time.Second

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	var results []requestResult

	for time.Since(startTime) < duration {
		reqStart := time.Now()
		resp, err := client.Get(url)

		if err != nil {
			results = append(results, requestResult{err: err})
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		latency := time.Since(reqStart)

		results = append(results, requestResult{
			latency:    latency,
			statusCode: resp.StatusCode,
		})
	}
	channel <- results
}
