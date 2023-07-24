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

func GetConfigForName(config_name string) ([]string, []string) {
	if len(os.Getenv(config_name+"_REPOSITORY")) > 0 && len(os.Getenv(config_name+"_BRANCH")) > 0 {
		return strings.Split(os.Getenv(config_name+"_REPOSITORY"), ","), strings.Split(os.Getenv(config_name+"_BRANCH"), ",")
	}
	return []string{}, []string{}
}

func GetRepositoryConfig() ([]string, []string, string, error) {

	repos, branches := GetConfigForName("GITHUB")
	if len(repos) > 0 && len(branches) > 0 {
		return repos, branches, "gh", nil
	}

	repos, branches = GetConfigForName("BITBUCKET")
	if len(repos) > 0 && len(branches) > 0 {
		return repos, branches, "bb", nil
	}

	return []string{}, []string{}, "", fmt.Errorf("Missing environment variables. " +
		"Define either GITHUB_REPOSITORY and GITHUB_BRANCH, or BITBUCKET_REPOSITORY and BITBUCKET_BRANCH")
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
