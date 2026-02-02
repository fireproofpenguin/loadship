# loadship

**HTTP load testing + Docker container monitoring in one tool.**

Loadship answers the question: "Is this release better or worse than the last one?"

Whilst there are plenty of great tools out there to run load tests against endpoints and lots of guides on setting up the perfect stack to test your endpoint, load results into a DB, get graphs going, etc - loadship aims to be a simple to use tool that answers the question "is this release better or worse than the last one"

> [!NOTE]
> loadship is an in progress project that aims to solve my specific use cases. Additionally this is my first go project released. PRs are welcome and encouraged.

## Installation

Download the [latest release](https://github.com/fireproofpenguin/loadship/releases/latest) for your platform.

## Usage

### Load test against endpoint
```bash
# Run a load test for 10s with 10 connections
loadship run https://httpbin.org -d 10s -c 10 
```

Output:
```text
Running test (10/10s)  100% |████████████████████████████████████████|  
Load test complete. Processing results...
Total Requests: 638
Successful Requests: 638
Failed Requests: 0
Requests per Second: 63.80
Latency Min/Avg/Max: 90 / 159.80 / 1170 ms
Latency p50/p90/p95/p99: 102 / 321 / 503 / 730 ms
```

### Load test with docker metrics and file output
```bash
loadship run http://localhost:8080 --container nginx -j baseline.json
```

Output:
```text
=== Request Metrics ===
Total Requests: 110210
Successful Requests: 110210
Failed Requests: 0
Requests per Second: 3673.67
Latency Min/Avg/Max: 0 / 2.32 / 50 ms
Latency p50/p90/p95/p99: 2 / 4 / 4 / 5 ms
=== Docker Metrics ===
Average memory: 19.56 MB
Min memory: 17.50 MB
Max memory: 20.55 MB

✓ Results saved to ./baseline.json
```

### Compare test runs
```bash
loadship compare ./baseline.json new_deploy.json
```

Output:
```text
=== Comparing Test Results ===
Baseline: .\baseline.json (2026-02-02 21:47:46.2996229 +0000 UTC)
Test: .\new_deploy.json (2026-02-02 21:47:55.7833856 +0000 UTC)

=== HTTP Metrics ===
Metric          Baseline Test   Change
------          ------   ------ ------
Total Requests  24       31     +7 (29.17%) ✓
Failed Requests 0        0      0 (0.00%)
RPS             4.80     6.20   +1.40 (29.17%) ✓
Latency (Avg)   213      162    -52 (-24.18%) ✓
Latency (p50)   95       94     -1 (-1.05%)
Latency (p90)   519      355    -164 (-31.60%) ✓
Latency (p95)   587      358    -229 (-39.01%) ✓
Latency (p99)   1126     761    -365 (-32.42%) ✓
```

## Why loadship?

- **All-in-one**: HTTP load testing + container resource monitoring
- **Zero setup**: Single binary, no databases or dashboards to configure
- **CI/CD friendly**: Compare releases automatically in your pipeline

## Roadmap

Currently basic HTTP load testing is available, however the plan is to expand the tool to offer the following features in future.

- [x] HTTP load testing with percentile latencies
- [-] Docker resource monitoring
    - [x] Memory utilisation
    - [ ] Other stats
- [x] Stat output for results
- [x] Comparison between runs
- [ ] HTML reports with graphs
- [ ] Config files for complex scenarios
- [ ] Advanced load patterns (ramps, spikes, etc)