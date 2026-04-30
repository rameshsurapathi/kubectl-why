package analyzer

import (
	"fmt"
	"sort"
)

// ServiceTraceResult is the v3 relationship view for Service traffic.
type ServiceTraceResult struct {
	SchemaVersion      string           `json:"schemaVersion"`
	Resource           string           `json:"resource"`
	Namespace          string           `json:"namespace"`
	Status             string           `json:"status"`
	Summary            []string         `json:"summary"`
	EndpointCount      int              `json:"endpointCount"`
	ReadyEndpointCount int              `json:"readyEndpointCount"`
	Service            AnalysisResult   `json:"service"`
	BackingPods        []AnalysisResult `json:"backingPods"`
}

// BuildServiceTraceResult combines Service and backing Pod diagnoses.
func BuildServiceTraceResult(
	service AnalysisResult,
	backingPods []AnalysisResult,
	endpointCount int,
	readyEndpointCount int,
) ServiceTraceResult {
	rankedPods := append([]AnalysisResult(nil), backingPods...)
	sort.SliceStable(rankedPods, func(i, j int) bool {
		left := severityRank(rankedPods[i].Severity)
		right := severityRank(rankedPods[j].Severity)
		if left != right {
			return left > right
		}
		return rankedPods[i].Resource < rankedPods[j].Resource
	})

	critical, warning := 0, 0
	for _, pod := range rankedPods {
		switch pod.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}

	status := "Healthy"
	switch {
	case service.Severity == "critical" || critical > 0 ||
		(len(rankedPods) > 0 && readyEndpointCount == 0):
		status = "Critical"
	case service.Severity == "warning" || warning > 0 ||
		(len(rankedPods) > 0 && readyEndpointCount < endpointCount):
		status = "Warning"
	}

	summary := []string{
		serviceSummary(service),
		pluralize(endpointCount, "endpoint") + " discovered",
		pluralize(readyEndpointCount, "ready endpoint"),
	}
	if critical > 0 || warning > 0 {
		if critical > 0 {
			summary = append(summary, fmt.Sprintf("Service is impacted because %d backing pods are in a critical state.", critical))
		} else {
			summary = append(summary, fmt.Sprintf("Service is degraded because %d backing pods have warnings.", warning))
		}
	}

	return ServiceTraceResult{
		SchemaVersion:      "v3",
		Resource:           service.Resource,
		Namespace:          service.Namespace,
		Status:             status,
		Summary:            summary,
		EndpointCount:      endpointCount,
		ReadyEndpointCount: readyEndpointCount,
		Service:            service,
		BackingPods:        rankedPods,
	}
}

func serviceSummary(service AnalysisResult) string {
	if len(service.Summary) > 0 {
		return service.Summary[0]
	}
	if service.PrimaryReason != "" {
		return service.PrimaryReason
	}
	return "Service diagnosis completed."
}

// DeploymentTraceResult is the v3 relationship view for Deployment workloads.
type DeploymentTraceResult struct {
	SchemaVersion string           `json:"schemaVersion"`
	Resource      string           `json:"resource"`
	Namespace     string           `json:"namespace"`
	Status        string           `json:"status"`
	Summary       []string         `json:"summary"`
	Deployment    AnalysisResult   `json:"deployment"`
	ReplicaSets   []AnalysisResult `json:"replicaSets"`
	Pods          []AnalysisResult `json:"pods"`
	Dependencies  []AnalysisResult `json:"dependencies"`
}

// BuildDeploymentTraceResult combines Deployment, ReplicaSets, Pods, and Dependencies diagnoses.
func BuildDeploymentTraceResult(
	deployment AnalysisResult,
	replicaSets []AnalysisResult,
	pods []AnalysisResult,
	dependencies []AnalysisResult,
) DeploymentTraceResult {
	rankedPods := append([]AnalysisResult(nil), pods...)
	sort.SliceStable(rankedPods, func(i, j int) bool {
		left := severityRank(rankedPods[i].Severity)
		right := severityRank(rankedPods[j].Severity)
		if left != right {
			return left > right
		}
		return rankedPods[i].Resource < rankedPods[j].Resource
	})

	rankedDeps := append([]AnalysisResult(nil), dependencies...)
	sort.SliceStable(rankedDeps, func(i, j int) bool {
		left := severityRank(rankedDeps[i].Severity)
		right := severityRank(rankedDeps[j].Severity)
		if left != right {
			return left > right
		}
		return rankedDeps[i].Resource < rankedDeps[j].Resource
	})

	critical, warning := 0, 0
	for _, p := range rankedPods {
		switch p.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}
	for _, d := range rankedDeps {
		switch d.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}
	for _, rs := range replicaSets {
		switch rs.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}

	status := "Healthy"
	switch {
	case deployment.Severity == "critical" || critical > 0:
		status = "Critical"
	case deployment.Severity == "warning" || warning > 0:
		status = "Warning"
	}

	summary := []string{
		serviceSummary(deployment),
		fmt.Sprintf("%d replica sets, %d pods, %d dependencies", len(replicaSets), len(pods), len(dependencies)),
	}

	return DeploymentTraceResult{
		SchemaVersion: "v3",
		Resource:      deployment.Resource,
		Namespace:     deployment.Namespace,
		Status:        status,
		Summary:       summary,
		Deployment:    deployment,
		ReplicaSets:   replicaSets,
		Pods:          rankedPods,
		Dependencies:  rankedDeps,
	}
}

// IngressTraceResult is the v3 relationship view for Ingress traffic.
type IngressTraceResult struct {
	SchemaVersion   string               `json:"schemaVersion"`
	Resource        string               `json:"resource"`
	Namespace       string               `json:"namespace"`
	Status          string               `json:"status"`
	Summary         []string             `json:"summary"`
	Ingress         AnalysisResult       `json:"ingress"`
	ServiceTraces   []ServiceTraceResult `json:"serviceTraces"`
}

// BuildIngressTraceResult combines Ingress and backing Service diagnoses.
func BuildIngressTraceResult(
	ingress AnalysisResult,
	serviceTraces []ServiceTraceResult,
) IngressTraceResult {
	critical, warning := 0, 0
	
	for _, st := range serviceTraces {
		switch st.Status {
		case "Critical":
			critical++
		case "Warning":
			warning++
		}
	}

	status := "Healthy"
	switch {
	case ingress.Severity == "critical" || critical > 0:
		status = "Critical"
	case ingress.Severity == "warning" || warning > 0:
		status = "Warning"
	}

	summary := []string{
		serviceSummary(ingress),
		fmt.Sprintf("Routes traffic to %d services", len(serviceTraces)),
	}

	return IngressTraceResult{
		SchemaVersion:   "v3",
		Resource:        ingress.Resource,
		Namespace:       ingress.Namespace,
		Status:          status,
		Summary:         summary,
		Ingress:         ingress,
		ServiceTraces:   serviceTraces,
	}
}
