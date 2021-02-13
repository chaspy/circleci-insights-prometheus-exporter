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
	wfSuccessRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "success_rate",
		Help:      "success rate of workflow",
	},
		[]string{"workflow", "repo", "branch"},
	)
	wfDurationMetricsMin = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_min",
		Help:      "minimum duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	wfDurationMetricsMax = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_max",
		Help:      "maximum duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	wfDurationMetricsMedian = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_median",
		Help:      "median of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	wfDurationMetricsP95 = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_p95",
		Help:      "95 percentile of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	wfDurationMetricsStandardDeviation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "workflow_insight",
		Name:      "duration_metrics_standard_deviation",
		Help:      "standard deviation of duration metrics",
	},
		[]string{"workflow", "repo", "branch"},
	)
	jobSuccessRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "success_rate",
		Help:      "success rate of workflow",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
	jobDurationMetricsMin = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "duration_metrics_min",
		Help:      "minimum duration metrics",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
	jobDurationMetricsMax = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "duration_metrics_max",
		Help:      "maximum duration metrics",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
	jobDurationMetricsMedian = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "duration_metrics_median",
		Help:      "median of duration metrics",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
	jobDurationMetricsP95 = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "duration_metrics_p95",
		Help:      "95 percentile of duration metrics",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
	jobDurationMetricsStandardDeviation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "circleci_custom",
		Subsystem: "job_insight",
		Name:      "duration_metrics_standard_deviation",
		Help:      "standard deviation of duration metrics",
	},
		[]string{"job", "workflow", "repo", "branch"},
	)
)

type WorkflowsWithRepo struct {
	repo     string
	workflow string
}

type WorkflowInsightWithRepo struct {
	repo   string
	branch string
	WorkflowInsight
}

type WorkflowInsight struct {
	NextPageToken string `json:"next_page_token"`
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

type WorkflowJobsInsightWithRepo struct {
	repo     string
	branch   string
	workflow string
	WorkflowJobsInsight
}

type WorkflowJobsInsight struct {
	NextPageToken string `json:"next_page_token"`
	Items         []struct {
		Name    string `json:"name"`
		Metrics struct {
			TotalRuns        int     `json:"total_runs"`
			SuccessfulRuns   int     `json:"successful_runs"`
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
			Throughput float64 `json:"throughput"`
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

	prometheus.MustRegister(wfSuccessRate)
	prometheus.MustRegister(wfDurationMetricsMin)
	prometheus.MustRegister(wfDurationMetricsMax)
	prometheus.MustRegister(wfDurationMetricsMedian)
	prometheus.MustRegister(wfDurationMetricsP95)
	prometheus.MustRegister(wfDurationMetricsStandardDeviation)

	prometheus.MustRegister(jobSuccessRate)
	prometheus.MustRegister(jobDurationMetricsMin)
	prometheus.MustRegister(jobDurationMetricsMax)
	prometheus.MustRegister(jobDurationMetricsMedian)
	prometheus.MustRegister(jobDurationMetricsP95)
	prometheus.MustRegister(jobDurationMetricsStandardDeviation)

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
	var workflowWithRepos []WorkflowsWithRepo

	wfSuccessRate.Reset()
	wfDurationMetricsMin.Reset()
	wfDurationMetricsMax.Reset()
	wfDurationMetricsMedian.Reset()
	wfDurationMetricsP95.Reset()
	wfDurationMetricsStandardDeviation.Reset()

	jobSuccessRate.Reset()
	jobDurationMetricsMin.Reset()
	jobDurationMetricsMax.Reset()
	jobDurationMetricsMedian.Reset()
	jobDurationMetricsP95.Reset()
	jobDurationMetricsStandardDeviation.Reset()

	wfInsightWithRepos, err := getV2WorkflowInsights()
	if err != nil {
		return fmt.Errorf("failed to get workflow insights: %w", err)
	}

	// Extract repository and workflow name
	for _, wfInsightWithRepo := range wfInsightWithRepos {
		for _, wfInsight := range wfInsightWithRepo.WorkflowInsight.Items {
			wfWithRepo := WorkflowsWithRepo{
				repo:     wfInsightWithRepo.repo,
				workflow: wfInsight.Name,
			}
			workflowWithRepos = append(workflowWithRepos, wfWithRepo)
		}
	}

	wfJobsInsightWithRepos, err := getV2WorkflowJobsInsights(workflowWithRepos)
	if err != nil {
		return fmt.Errorf("failed to get workflow jobs insights: %w", err)
	}

	for _, wfInsightWithRepo := range wfInsightWithRepos {
		for _, wfInsight := range wfInsightWithRepo.WorkflowInsight.Items {
			labels := prometheus.Labels{
				"workflow": wfInsight.Name,
				"repo":     wfInsightWithRepo.repo,
				"branch":   wfInsightWithRepo.branch,
			}
			wfSuccessRate.With(labels).Set(wfInsight.Metrics.SuccessRate)
			wfDurationMetricsMin.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Min))
			wfDurationMetricsMax.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Max))
			wfDurationMetricsMedian.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.Median))
			wfDurationMetricsP95.With(labels).Set(float64(wfInsight.Metrics.DurationMetrics.P95))
			wfDurationMetricsStandardDeviation.With(labels).Set(wfInsight.Metrics.DurationMetrics.StandardDeviation)
		}
	}

	for _, wfJobsInsightWithRepo := range wfJobsInsightWithRepos {
		for _, wfJobsInsight := range wfJobsInsightWithRepo.WorkflowJobsInsight.Items {
			labels := prometheus.Labels{
				"job":      wfJobsInsight.Name,
				"workflow": wfJobsInsightWithRepo.workflow,
				"repo":     wfJobsInsightWithRepo.repo,
				"branch":   wfJobsInsightWithRepo.branch,
			}
			jobSuccessRate.With(labels).Set(wfJobsInsight.Metrics.SuccessRate)
			jobDurationMetricsMin.With(labels).Set(float64(wfJobsInsight.Metrics.DurationMetrics.Min))
			jobDurationMetricsMax.With(labels).Set(float64(wfJobsInsight.Metrics.DurationMetrics.Max))
			jobDurationMetricsMedian.With(labels).Set(float64(wfJobsInsight.Metrics.DurationMetrics.Median))
			jobDurationMetricsP95.With(labels).Set(float64(wfJobsInsight.Metrics.DurationMetrics.P95))
			jobDurationMetricsStandardDeviation.With(labels).Set(wfJobsInsight.Metrics.DurationMetrics.StandardDeviation)
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
	var pageToken string

	reportingWindow := getReportingWindow()
	repos, err := getGitHubRepos()
	if err != nil {
		return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to read GitHub repository: %w", err)
	}

	branches, err := getGitHubBranches()
	if err != nil {
		return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to read GitHub branch: %w", err)
	}

	getCircleCIToken, err := getCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	for _, repo := range repos {
		for _, branch := range branches {
			for {
				url := "https://circleci.com/api/v2/insights/gh/" + repo + "/workflows?" + "&branch=" + branch + "&reporting-window=" + reportingWindow + "&page-token" + pageToken

				ctx := context.Background()
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

				req.Header.Add("Circle-Token", getCircleCIToken)

				body, status, err := getV2WorkflowInsightsAPI(req)
				if status >= 300 { //nolint:gomnd
					log.Printf("response status code is not 2xx. status code: %v body: %v. skip.\n", status, string(body))
					return nil, nil
				}

				if err != nil {
					return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to call API %w", err)
				}

				err = json.Unmarshal(body, &wfInsight)
				if err != nil {
					return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to parse response body. body %v, err %w", string(body), err)
				}

				wfInsightWithRepo := WorkflowInsightWithRepo{repo: repo, branch: branch, WorkflowInsight: wfInsight}
				wfInsightWithRepos = append(wfInsightWithRepos, wfInsightWithRepo)

				// pagination
				if wfInsight.NextPageToken == "" {
					break
				} else {
					pageToken = wfInsight.NextPageToken
				}
			}
		}
	}
	return wfInsightWithRepos, nil
}

func getV2WorkflowInsightsAPI(req *http.Request) ([]byte, int, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get response body: %w", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, res.StatusCode, nil
}

func getV2WorkflowJobsInsights(workflowWithRepos []WorkflowsWithRepo) ([]WorkflowJobsInsightWithRepo, error) {
	var wfJobsInsight WorkflowJobsInsight
	var wfJobsInsightWithRepos []WorkflowJobsInsightWithRepo
	var pageToken string

	reportingWindow := getReportingWindow()

	branches, err := getGitHubBranches()
	if err != nil {
		return []WorkflowJobsInsightWithRepo{}, fmt.Errorf("failed to read GitHub branch: %w", err)
	}

	getCircleCIToken, err := getCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	for _, workflowWithRepo := range workflowWithRepos {
		for _, branch := range branches {
			for {
				url := "https://circleci.com/api/v2/insights/gh/" + workflowWithRepo.repo + "/workflows/" + workflowWithRepo.workflow + "/jobs" + "?branch=" + branch + "&reporting-window=" + reportingWindow + "&page-token" + pageToken

				ctx := context.Background()
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

				req.Header.Add("Circle-Token", getCircleCIToken)

				body, status, err := getV2WorkflowJobsInsightsAPI(req)
				if status >= 300 { //nolint:gomnd
					log.Printf("response status code is not 2xx. status code: %v body: %v. skip.\n", status, string(body))
					return []WorkflowJobsInsightWithRepo{}, nil
				}

				if err != nil {
					return []WorkflowJobsInsightWithRepo{}, fmt.Errorf("failed to call API %w", err)
				}

				err = json.Unmarshal(body, &wfJobsInsight)
				if err != nil {
					return []WorkflowJobsInsightWithRepo{}, fmt.Errorf("failed to parse response body. body %v, err %w", string(body), err)
				}

				wfJobsInsightWithRepo := WorkflowJobsInsightWithRepo{repo: workflowWithRepo.repo, branch: branch, workflow: workflowWithRepo.workflow, WorkflowJobsInsight: wfJobsInsight}
				wfJobsInsightWithRepos = append(wfJobsInsightWithRepos, wfJobsInsightWithRepo)

				// pagination
				if wfJobsInsight.NextPageToken == "" {
					break
				} else {
					pageToken = wfJobsInsight.NextPageToken
				}
			}
		}
	}

	return wfJobsInsightWithRepos, nil
}

func getV2WorkflowJobsInsightsAPI(req *http.Request) ([]byte, int, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get response body: %w", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, res.StatusCode, nil
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

// reporting window expect the followings:
// "last-7-days" "last-90-days" "last-24-hours" "last-30-days" "last-60-days"
// ref:https://circleci.com/docs/api/v2/#tag/Insights
func getReportingWindow() string {
	defaultReportingWindow := "last-7-days"
	reportingWindow := os.Getenv("REPORTING_WINDOW")
	if len(reportingWindow) == 0 {
		return defaultReportingWindow
	}

	return reportingWindow
}
