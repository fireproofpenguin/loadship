package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/go-sdk/client"
	mobyContainer "github.com/moby/moby/api/types/container"
	moby "github.com/moby/moby/client"
)

type DockerStats struct {
	MemoryUsageMB float64
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

		stat := DockerStats{
			MemoryUsageMB: memoryMB,
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
