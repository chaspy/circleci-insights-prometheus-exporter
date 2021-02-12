package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	getV2WorkflowInsights()
//	getV2InsightWorkflowJob()
}

func getV2WorkflowInsights() {
	type workflowInsight struct{
		name string
		total_runs int64
		successful_runs int64
		mttr int64
		total_credits_used int64
		failed_runs int64
		success_rate float64
		duration_metrics_min int64
		duration_metrics_max int64
		duration_metrics_median int64
		duration_metrics_mean int64
		duration_metrics_p95 int64
		duration_metrics_standard_deviation float64
		total_recoveries int64
		throughput float64
	}

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

	//	fmt.Println(res)
	fmt.Println(string(body))
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