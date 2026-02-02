package load

import (
	"context"
	"io"
	"net/http"
	"strings"
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
			errorType := classifyError(err)
			results = append(results, HTTPStats{Timestamp: reqStart, ErrorType: errorType})
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

func classifyError(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "connection refused") {
		return "connection_refused"
	}
	if strings.Contains(errStr, "timeout") {
		return "timeout"
	}
	if strings.Contains(errStr, "no such host") {
		return "dns_error"
	}
	if strings.Contains(errStr, "EOF") {
		return "connection_reset"
	}

	return "unknown" // Catch-all
}
