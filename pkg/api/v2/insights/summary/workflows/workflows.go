package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/chaspy/circleci-insight-prometheus-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
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
)

type WorkflowWithRepo struct {
	Repo     string
	Workflow string
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

func Register() {
	prometheus.MustRegister(wfSuccessRate)
	prometheus.MustRegister(wfDurationMetricsMin)
	prometheus.MustRegister(wfDurationMetricsMax)
	prometheus.MustRegister(wfDurationMetricsMedian)
	prometheus.MustRegister(wfDurationMetricsP95)
	prometheus.MustRegister(wfDurationMetricsStandardDeviation)
}

func Export() ([]WorkflowWithRepo, error) {
	var workflowWithRepos []WorkflowWithRepo

	wfSuccessRate.Reset()
	wfDurationMetricsMin.Reset()
	wfDurationMetricsMax.Reset()
	wfDurationMetricsMedian.Reset()
	wfDurationMetricsP95.Reset()
	wfDurationMetricsStandardDeviation.Reset()

	wfInsightWithRepos, err := getV2WorkflowInsights()
	if err != nil {
		return []WorkflowWithRepo{}, fmt.Errorf("failed to get workflow insights: %w", err)
	}

	// Extract repository and workflow name
	for _, wfInsightWithRepo := range wfInsightWithRepos {
		for _, wfInsight := range wfInsightWithRepo.WorkflowInsight.Items {
			wfWithRepo := WorkflowWithRepo{
				Repo:     wfInsightWithRepo.repo,
				Workflow: wfInsight.Name,
			}
			workflowWithRepos = append(workflowWithRepos, wfWithRepo)
		}
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

	return workflowWithRepos, nil
}

func getV2WorkflowInsights() ([]WorkflowInsightWithRepo, error) {
	var wfInsight WorkflowInsight
	var wfInsightWithRepos []WorkflowInsightWithRepo
	var pageToken string

	reportingWindow := config.GetReportingWindow()
	repos, err := config.GetGitHubRepos()
	if err != nil {
		return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to read GitHub repository: %w", err)
	}

	branches, err := config.GetGitHubBranches()
	if err != nil {
		return []WorkflowInsightWithRepo{}, fmt.Errorf("failed to read GitHub branch: %w", err)
	}

	getCircleCIToken, err := config.GetCircleCIToken()
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
