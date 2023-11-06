package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/chaspy/circleci-insights-prometheus-exporter/pkg/api/v2/insights/summary/jobs"
	"github.com/chaspy/circleci-insights-prometheus-exporter/pkg/api/v2/insights/summary/workflows"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	interval, err := getInterval()
	if err != nil {
		log.Fatal(err)
	}

	workflows.Register()
	jobs.Register()

	http.Handle("/metrics", promhttp.Handler())

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			// If you wish to keep the "old" way of count time and then work - replace "continue" down below with the function run
			err := snapshot()
			if err != nil {
				log.Fatal(err)
			}

			select {
				case <-ticker.C:
					continue
				case <-interrupt:
					ticker.Stop()
					return
				}
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func snapshot() error {
	workflowsWithRepo, err := workflows.Export()
	if err != nil {
		return fmt.Errorf("failed to export workflow insights metrics: %w", err)
	}

	err = jobs.Export(workflowsWithRepo)
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
