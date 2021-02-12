package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//nolint:gochecknoglobals
var (
	successRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "success_rate",
		Help:      "success rate of workflow",
	},
		[]string{"workflow", "repo", "branch"},
	)
	durationMetricsMin = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_min",
		Help:      "minimum duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	durationMetricsMax = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_max",
		Help:      "maximum duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	durationMetricsMedian = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_median",
		Help:      "median of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	durationMetricsP95 = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_p95",
		Help:      "95 percentile of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	durationMetricsStandardDeviation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_standard_deviation",
		Help:      "standard deviation of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
)

type WorkflowInsightWithRepo struct {
	repo   string
	branch string
	WorkflowInsight
}

type WorkflowInsight struct {
	NextPageToken interface{} `json:"next_page_token"`
	Items         []struct {
		Name    string `json:"name"`
		Metrics struct {
			TotalRuns        int     `json:"total_runs"`
			SuccessfulRuns   int     `json:"successful_runs"`
			Mttr             int     `json:"mttr"`
			TotalCreditsUsed int     `json:"total_credits_used"`
			FailedRuns       int     `json:"failed_runs"`
			SuccessRate      float64 `json:"success_rate"`
			DurationMetrics  struct {
				Min               int     `json:"min"`
				Max               int     `json:"max"`
				Median            int     `json:"median"`
				Mean              int     `json:"mean"`
				P95               int     `json:"p95"`
				StandardDeviation float64 `json:"standard_deviation"`
			} `json:"duration_metrics"`
			TotalRecoveries int     `json:"total_recoveries"`
			Throughput      float64 `json:"throughput"`
		} `json:"metrics"`
		WindowStart time.Time `json:"window_start"`
		WindowEnd   time.Time `json:"window_end"`
	} `json:"items"`
}

func main() {
	interval, err := getInterval()
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(successRate)
	prometheus.MustRegister(durationMetricsMin)
	prometheus.MustRegister(durationMetricsMax)
	prometheus.MustRegister(durationMetricsMedian)
	prometheus.MustRegister(durationMetricsP95)
	prometheus.MustRegister(durationMetricsStandardDeviation)

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
	successRate.Reset()
	durationMetricsMin.Reset()
	durationMetricsMax.Reset()
	durationMetricsMedian.Reset()
	durationMetricsP95.Reset()
	durationMetricsStandardDeviation.Reset()

	wfInsightWithRepos, err := getV2WorkflowInsights()
	if err != nil {
		return fmt.Errorf("failed to get workflow insights: %w", err)
	}

	for _, wfInsightWithRepo := range wfInsightWithRepos {
		for _, wfInsight := range wfInsightWithRepo.WorkflowInsight.Items {
			labels := prometheus.Labels{
				"workflow": wfInsight.Name,
				"repo":     wfInsightWithRepo.repo,
				"branch":   wfInsightWithRepo.branch,
			}
			successRate.With(labels).Set(wfInsight.Metrics.SuccessRate)
			durationMetricsMin.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Min))
			durationMetricsMax.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Max))
			durationMetricsMedian.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Median))
			durationMetricsP95.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.P95))
			durationMetricsStandardDeviation.With(labels).Set(wfInsight.Metrics.DurationMetrics.StandardDeviation)
		}
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

func getV2WorkflowInsights() ([]WorkflowInsightWithRepo, error) {
	var wfInsight WorkflowInsight
	var wfInsightWithRepos []WorkflowInsightWithRepo

	reportingWingow := "last-7-days"
	repos, err := getGitHubRepos()
	branches, err := getGitHubBranches()

	getCircleCIToken, err := getCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	for _, repo := range repos {
		for _, branch := range branches {
			fmt.Printf("%v,%v\n", repo, branch)
			// TODO: pagination
			// if next_page_token is nil, break.
			// otherwise, set the token to "page-token" query parameter
			// ref: https://circleci.com/docs/api/v2/?utm_medium=SEM&utm_source=gnb&utm_campaign=SEM-gb-DSA-Eng-japac&utm_content=&utm_term=dynamicSearch-&gclid=CjwKCAiA65iBBhB-EiwAW253W3odzDASJ4KM0jAwNejVKqmjFz5a_74x8oIGy5jGm_MUZkhqnmtFkhoC7QIQAvD_BwE#operation/getProjectWorkflowMetrics

			url := "https://circleci.com/api/v2/insights/gh/" + repo + "/workflows?" + "&branch=" + branch + "&reporting-window=" + reportingWingow

			ctx := context.Background()
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

			req.Header.Add("Circle-Token", getCircleCIToken)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to get response body: %w", err)
			}

			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to read response body: %w", err)
			}
			fmt.Println(string(body))
			err = json.Unmarshal(body, &wfInsight)
			if err != nil {
				return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to parse response body: %w", err)
			}

			wfInsightWithRepo := WorkflowInsightWithRepo{repo: repo, branch: branch, WorkflowInsight: wfInsight}
			wfInsightWithRepos = append(wfInsightWithRepos, wfInsightWithRepo)
		}
	}
	return wfInsightWithRepos, nil
}

func getCircleCIToken() (string, error) {
	circleciToken := os.Getenv("CIRCLECI_TOKEN")
	if len(circleciToken) == 0 {
		return "", fmt.Errorf("missing environment variable CIRCLECI_TOKEN")
	}

	return circleciToken, nil
}

func getGitHubRepos() ([]string, error) {
	githubRepos := os.Getenv("GITHUB_REPOSITORY")
	if len(githubRepos) == 0 {
		return []string{}, fmt.Errorf("missing environment variable GITHUB_REPOSITORY")
	}
	ret := strings.Split(githubRepos, ",")
	return ret, nil
}

func getGitHubBranches() ([]string, error) {
	githubBranches := os.Getenv("GITHUB_BRANCH")
	if len(githubBranches) == 0 {
		return []string{}, fmt.Errorf("missing environment variable GITHUB_BRANCH")
	}
	ret := strings.Split(githubBranches, ",")
	return ret, nil
}
