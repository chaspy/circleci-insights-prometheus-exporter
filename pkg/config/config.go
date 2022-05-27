package config

import (
	"fmt"
	"os"
	"strings"
)

func GetCircleCIToken() (string, error) {
	circleciToken := os.Getenv("CIRCLECI_TOKEN")
	if len(circleciToken) == 0 {
		return "", fmt.Errorf("missing environment variable CIRCLECI_TOKEN")
	}

	return circleciToken, nil
}

func GetGitHubRepos() ([]string, error) {
	githubRepos := os.Getenv("GITHUB_REPOSITORY")
	if len(githubRepos) == 0 {
		return []string{}, fmt.Errorf("missing environment variable GITHUB_REPOSITORY")
	}
	ret := strings.Split(githubRepos, ",")
	return ret, nil
}

func GetGitHubBranches() ([]string, error) {
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
func GetReportingWindow() string {
	defaultReportingWindow := "last-7-days"
	reportingWindow := os.Getenv("REPORTING_WINDOW")
	if len(reportingWindow) == 0 {
		return defaultReportingWindow
	}

	return reportingWindow
}
