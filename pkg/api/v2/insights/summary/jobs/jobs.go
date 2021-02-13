package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/chaspy/circleci-insight-prometheus-exporter/pkg/api/v2/insights/summary/workflows"
	"github.com/chaspy/circleci-insight-prometheus-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

//nolint:gochecknoglobals
var (
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

func Register() {
	prometheus.MustRegister(jobSuccessRate)
	prometheus.MustRegister(jobDurationMetricsMin)
	prometheus.MustRegister(jobDurationMetricsMax)
	prometheus.MustRegister(jobDurationMetricsMedian)
	prometheus.MustRegister(jobDurationMetricsP95)
	prometheus.MustRegister(jobDurationMetricsStandardDeviation)
}

func Export(workflowWithRepos []workflows.WorkflowWithRepo) error {
	jobSuccessRate.Reset()
	jobDurationMetricsMin.Reset()
	jobDurationMetricsMax.Reset()
	jobDurationMetricsMedian.Reset()
	jobDurationMetricsP95.Reset()
	jobDurationMetricsStandardDeviation.Reset()

	wfJobsInsightWithRepos, err := getV2WorkflowJobsInsights(workflowWithRepos)
	if err != nil {
		return fmt.Errorf("failed to get workflow jobs insights: %w", err)
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

func getV2WorkflowJobsInsights(workflowWithRepos []workflows.WorkflowWithRepo) ([]WorkflowJobsInsightWithRepo, error) {
	var wfJobsInsight WorkflowJobsInsight
	var wfJobsInsightWithRepos []WorkflowJobsInsightWithRepo
	var pageToken string

	reportingWindow := config.GetReportingWindow()

	branches, err := config.GetGitHubBranches()
	if err != nil {
		return []WorkflowJobsInsightWithRepo{}, fmt.Errorf("failed to read GitHub branch: %w", err)
	}

	getCircleCIToken, err := config.GetCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	for _, workflowWithRepo := range workflowWithRepos {
		for _, branch := range branches {
			for {
				url := "https://circleci.com/api/v2/insights/gh/" + workflowWithRepo.Repo + "/workflows/" + workflowWithRepo.Workflow + "/jobs" + "?branch=" + branch + "&reporting-window=" + reportingWindow + "&page-token" + pageToken

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

				wfJobsInsightWithRepo := WorkflowJobsInsightWithRepo{repo: workflowWithRepo.Repo, branch: branch, workflow: workflowWithRepo.Workflow, WorkflowJobsInsight: wfJobsInsight}
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
