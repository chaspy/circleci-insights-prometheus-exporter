# circleci-insight-prometheus-exporter

Prometheus Exporter for [CircleCI Insight API](https://circleci.com/docs/api/v2/#tag/Insights)

## How to run

### Local

```
$ go run main.go
```

### Binary

Get the binary file from [Releases](https://github.com/chaspy/circleci-insight-prometheus-exporter/releases) and run it.

### Docker

```
$ docker run chaspy/circleci-insight-prometheus-exporter:v0.1.0
```

## Metrics

### Get summary metrics for a project's workflows

* circleci_custom_workflow_insight_success_rate
* circleci_custom_workflow_insight_duration_metrics_min
* circleci_custom_workflow_insight_duration_metrics_max
* circleci_custom_workflow_insight_duration_metrics_median
* circleci_custom_workflow_insight_duration_metrics_p95
* circleci_custom_workflow_insight_duration_metrics_standard_deviation

These metrics are from [getProjectWorkflowRuns API](https://circleci.com/docs/api/v2/#operation/getProjectWorkflowRuns)

## Environment Variable

|name                 |required|default |description|
|---------------------|--------|--------|-----------|
|CIRCLECI_TOKEN       |yes     |-       |[CircleCI API Token](https://app.circleci.com/settings/user/tokens)|
|CIRCLECI_API_INTERVAL|no      |300(sec)|Interval second for calling the API|
|GITHUB_REPOSITORY    |yes     |-       |Comma-separated repository names. i.e. "chaspy/chaspy.me,chaspy/dotfiles"|
|GITHUB_BRANCH        |yes     |-       |Comma-separated branch names. i.e. "master,develop"|

## Datadog Autodiscovery

If you use Datadog, you can use [Kubernetes Integration Autodiscovery](https://docs.datadoghq.com/agent/kubernetes/integrations/?tab=kubernetes) feature.
