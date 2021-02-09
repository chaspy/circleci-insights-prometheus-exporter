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
}

func getV2WorkflowInsights() {
	branch := "develop"
	reportingWingow := "last-7-days"
	org := "quipper"
	repo := "monorepo"

	getCircleCIToken, err := getCircleCIToken()
	if err != nil {
		log.Fatal("failed to read Datadog Config: %w", err)
	}

	url := "https://circleci.com/api/v2/insights/gh/" + org + "/" + repo + "/workflows?" + "&branch=" + branch + "&reporting-window=" + reportingWingow

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", getCircleCIToken)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//	fmt.Println(res)
	fmt.Println(string(body))
}

func getCircleCIToken() (string, error) {
	circleciToken := os.Getenv("CIRCLECI_TOKEN")
	if len(circleciToken) == 0 {
		return "", fmt.Errorf("missing environment variable CIRCLECI_TOKEN")
	}

	return circleciToken, nil
}