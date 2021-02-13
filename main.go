package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	v2_workflow_insights "github.com/chaspy/circleci-insight-prometheus-exporter/pkg/V2WorkflowInsights"
	v2_workflow_jobs_insights "github.com/chaspy/circleci-insight-prometheus-exporter/pkg/V2WorkflowJobsInsights"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	interval, err := getInterval()
	if err != nil {
		log.Fatal(err)
	}

	v2_workflow_insights.Register()
	v2_workflow_jobs_insights.Register()

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)

		// register metrics as background
		for range ticker.C {
			err := snapshot()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func snapshot() error {
	workflowsWithRepo, err := v2_workflow_insights.Export()
	if err != nil {
		return fmt.Errorf("failed to export workflow insights metrics: %w", err)
	}

	err = v2_workflow_jobs_insights.Export(workflowsWithRepo)
	if err != nil {
		return fmt.Errorf("failed to export job insights metrics: %w", err)
	}

	return nil
}

func getInterval() (int, error) {
	const defaultCircleCIAPIIntervalSecond = 300
	circleciAPIInterval := os.Getenv("CIRCLECI_API_INTERVAL")
	if len(circleciAPIInterval) == 0 {
		return defaultCircleCIAPIIntervalSecond, nil
	}

	integerCircleCIAPIInterval, err := strconv.Atoi(circleciAPIInterval)
	if err != nil {
		return 0, fmt.Errorf("failed to read CircleCI Config: %w", err)
	}

	return integerCircleCIAPIInterval, nil
}
