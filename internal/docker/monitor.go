package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/docker/go-sdk/client"
	mobyContainer "github.com/moby/moby/api/types/container"
	moby "github.com/moby/moby/client"
)

type DockerStats struct {
	MemoryUsageMB float64
}

func RunDockerMonitor(ctx context.Context, container string) []DockerStats {
	log.SetPrefix("RunDockerMonitor")
	var results []DockerStats

	cli, err := client.New(context.Background())

	if err != nil {
		log.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	dockerCtx, dockerCtxCancel := context.WithCancel(context.Background())
	defer dockerCtxCancel()

	options := moby.ContainerStatsOptions{
		Stream: true,
	}

	stats, err := cli.ContainerStats(dockerCtx, container, options)

	if err != nil {
		log.Printf("Warning: Docker monitoring failed: %v\n", err)
		return nil
	}

	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)

	for {
		if ctx.Err() != nil {
			dockerCtxCancel()
			return results
		}

		response := mobyContainer.StatsResponse{}
		if err := decoder.Decode(&response); err != nil {
			dockerCtxCancel()
			return results
		}

		memoryMB := float64(response.MemoryStats.Usage) / 1024 / 1024

		stat := DockerStats{
			MemoryUsageMB: memoryMB,
		}
		results = append(results, stat)
	}
}

func CheckContainerRunning(container_id string) (bool, error) {
	cli, err := client.New(context.Background())

	if err != nil {
		return false, fmt.Errorf("failed to create docker client: %w", err)
	}

	defer cli.Close()

	inspect, err := cli.ContainerInspect(context.Background(), container_id, moby.ContainerInspectOptions{Size: false})

	if err != nil {
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}

	return inspect.Container.State.Running, nil
}
