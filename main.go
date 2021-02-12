package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	getV2WorkflowInsights()
//	getV2InsightWorkflowJob()
}

func getV2WorkflowInsights() {
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

	var m WorkflowInsight

	branch := "develop"
	reportingWingow := "last-7-days"
	org := "quipper"
	repo := "monorepo"

	getCircleCIToken, err := getCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	// TODO: pagination
	// if next_page_token is nil, break.
	// otherwise, set the token to "page-token" query parameter
	// ref: https://circleci.com/docs/api/v2/?utm_medium=SEM&utm_source=gnb&utm_campaign=SEM-gb-DSA-Eng-japac&utm_content=&utm_term=dynamicSearch-&gclid=CjwKCAiA65iBBhB-EiwAW253W3odzDASJ4KM0jAwNejVKqmjFz5a_74x8oIGy5jGm_MUZkhqnmtFkhoC7QIQAvD_BwE#operation/getProjectWorkflowMetrics

	url := "https://circleci.com/api/v2/insights/gh/" + org + "/" + repo + "/workflows?" + "&branch=" + branch + "&reporting-window=" + reportingWingow

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", getCircleCIToken)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(body, &m)
	fmt.Printf("%+v\n",m)
}

//func getV2InsightWorkflowJob(){
	// not implemented
//	fmt.Println("Hello insight workflow job")
//}

func getCircleCIToken() (string, error) {
	circleciToken := os.Getenv("CIRCLECI_TOKEN")
	if len(circleciToken) == 0 {
		return "", fmt.Errorf("missing environment variable CIRCLECI_TOKEN")
	}

	return circleciToken, nil
}