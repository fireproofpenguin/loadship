package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/go-sdk/client"
	mobyContainer "github.com/moby/moby/api/types/container"
	moby "github.com/moby/moby/client"
)

type DockerStats struct {
	Timestamp     time.Time `json:"timestamp"`
	MemoryUsageMB float64   `json:"memory_usage_mb"`
	CPUPercent    float64   `json:"cpu_percent"`
	DiskReadMB    float64   `json:"disk_read_mb"`
	DiskWriteMB   float64   `json:"disk_write_mb"`
	PIDs          uint64    `json:"pids"`
}

func RunDockerMonitor(ctx context.Context, container string) ([]DockerStats, error) {
	var results []DockerStats

	cli, err := client.New(context.Background())

	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	defer cli.Close()

	dockerCtx, dockerCtxCancel := context.WithCancel(context.Background())
	defer dockerCtxCancel()

	options := moby.ContainerStatsOptions{
		Stream: true,
	}

	stats, err := cli.ContainerStats(dockerCtx, container, options)

	if err != nil {
		return nil, fmt.Errorf("docker monitoring failed: %w", err)
	}

	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)

	var prevCPU, prevSystem uint64
	for {
		if ctx.Err() != nil {
			dockerCtxCancel()
			return results, nil
		}

		response := mobyContainer.StatsResponse{}
		if err := decoder.Decode(&response); err != nil {
			dockerCtxCancel()
			return results, nil
		}

		usage := response.MemoryStats.Usage
		inactiveFile := response.MemoryStats.Stats["inactive_file"]

		var workingSet uint64
		if usage > inactiveFile {
			workingSet = usage - inactiveFile
		} else {
			workingSet = usage
		}

		memoryMB := float64(workingSet) / 1024 / 1024

		cpuDelta := float64(response.CPUStats.CPUUsage.TotalUsage - prevCPU)
		systemDelta := float64(response.CPUStats.SystemUsage - prevSystem)
		prevCPU = response.CPUStats.CPUUsage.TotalUsage
		prevSystem = response.CPUStats.SystemUsage

		var cpuPercent float64
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * float64(response.CPUStats.OnlineCPUs) * 100.0
		}

		var diskReadBytes, diskWriteBytes uint64
		for _, stat := range response.BlkioStats.IoServiceBytesRecursive {
			switch stat.Op {
			case "read":
				diskReadBytes += stat.Value
			case "write":
				diskWriteBytes += stat.Value
			}
		}

		diskReadMB := float64(diskReadBytes) / 1024 / 1024
		diskWriteMB := float64(diskWriteBytes) / 1024 / 1024

		stat := DockerStats{
			Timestamp:     response.Read,
			MemoryUsageMB: memoryMB,
			CPUPercent:    cpuPercent,
			DiskReadMB:    diskReadMB,
			DiskWriteMB:   diskWriteMB,
			PIDs:          response.PidsStats.Current,
		}
		results = append(results, stat)
	}
}

func CheckContainerRunning(containerID string) (bool, error) {
	cli, err := client.New(context.Background())

	if err != nil {
		return false, fmt.Errorf("failed to create docker client: %w", err)
	}

	defer cli.Close()

	inspect, err := cli.ContainerInspect(context.Background(), containerID, moby.ContainerInspectOptions{Size: false})

	if err != nil {
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}

	return inspect.Container.State.Running, nil
}
