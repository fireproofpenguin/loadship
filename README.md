# loadship

**HTTP load testing + Docker container monitoring in one tool.**

Loadship answers the question: "Is this release better or worse than the last one?"

Whilst there are plenty of great tools out there to run load tests against endpoints and lots of guides on setting up the perfect stack to test your endpoint, load results into a DB, get graphs going, etc - loadship aims to be a simple to use tool that answers the question "is this release better or worse than the last one"

> [!NOTE]
> loadship is an in progress project that aims to solve my specific use cases. Additionally this is my first go project released. PRs are welcome and encouraged.

## Installation

Download the [latest release](https://github.com/fireproofpenguin/loadship/releases/latest) for your platform.

## Usage

```bash
# Run a load test for 10s with 10 connections
loadship run https://httpbin.org -d 10s -c 10 
```

Output:
```
Running test (10/10s)  100% |████████████████████████████████████████|  
Load test complete. Processing results...
Total Requests: 638
Successful Requests: 638
Failed Requests: 0
Requests per Second: 63.80
Latency Min/Avg/Max: 90 / 159.80 / 1170 ms
Latency p50/p90/p95/p99: 102 / 321 / 503 / 730 ms
```

## Why loadship?

- **All-in-one**: HTTP load testing + container resource monitoring
- **Zero setup**: Single binary, no databases or dashboards to configure
- **CI/CD friendly**: Compare releases automatically in your pipeline

## Roadmap

Currently basic HTTP load testing is available, however the plan is to expand the tool to offer the following features in future.

- [x] HTTP load testing with percentile latencies
- [ ] Docker resource monitoring
- [ ] Stat output for results
- [ ] Comparison between runs
- [ ] HTML reports with graphs
- [ ] Config files for complex scenarios
- [ ] Advanced load patterns (ramps, spikes, etc)