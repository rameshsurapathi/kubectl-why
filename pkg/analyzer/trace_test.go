package analyzer_test

import (
	"testing"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/stretchr/testify/assert"
)

func TestBuildServiceTraceResult_RanksBackingPodIssues(t *testing.T) {
	result := analyzer.BuildServiceTraceResult(
		analyzer.AnalysisResult{
			Resource:      "service/api",
			Namespace:     "default",
			PrimaryReason: "SOME_PODS_FAILING",
			Severity:      "warning",
			Summary:       []string{"Service matches 2 pods (1 ready, 1 failing)."},
		},
		[]analyzer.AnalysisResult{
			{
				Resource:      "pod/api-good",
				PrimaryReason: "Healthy",
				Severity:      "healthy",
			},
			{
				Resource:      "pod/api-bad",
				PrimaryReason: "Cannot pull container image from registry",
				Severity:      "critical",
				Findings: []analyzer.Finding{
					{ReasonCode: "IMAGE_PULL_FAILED", Confidence: "high"},
				},
			},
		},
		2,
		1,
	)

	assert.Equal(t, "v3", result.SchemaVersion)
	assert.Equal(t, "Critical", result.Status)
	assert.Equal(t, "service/api", result.Resource)
	assert.Equal(t, 2, result.EndpointCount)
	assert.Equal(t, 1, result.ReadyEndpointCount)
	assert.Equal(t, "pod/api-bad", result.BackingPods[0].Resource)
	assert.Contains(t, result.Summary, "Service is impacted because 1 backing pods are in a critical state.")
}

func TestBuildServiceTraceResult_ManualEndpointService(t *testing.T) {
	result := analyzer.BuildServiceTraceResult(
		analyzer.AnalysisResult{
			Resource:      "service/kubernetes",
			Namespace:     "default",
			PrimaryReason: "MANUAL_ENDPOINTS",
			Severity:      "healthy",
			Summary:       []string{"Service has no selector but has 1 manually managed endpoints."},
		},
		nil,
		1,
		1,
	)

	assert.Equal(t, "Healthy", result.Status)
	assert.Empty(t, result.BackingPods)
	assert.Contains(t, result.Summary, "1 endpoint discovered")
}

func TestBuildServiceTraceResult_ExternalNameWithoutEndpointsIsNotCritical(t *testing.T) {
	result := analyzer.BuildServiceTraceResult(
		analyzer.AnalysisResult{
			Resource:      "service/api-external",
			Namespace:     "default",
			PrimaryReason: "Service proxies to an external DNS name",
			Severity:      "info",
			Summary:       []string{"This service directs traffic to external name: api.example.com."},
		},
		nil,
		0,
		0,
	)

	assert.Equal(t, "Healthy", result.Status)
	assert.Empty(t, result.BackingPods)
}

func TestBuildDeploymentTraceResult_RanksPodsAndDependencies(t *testing.T) {
	result := analyzer.BuildDeploymentTraceResult(
		analyzer.AnalysisResult{
			Resource:      "deployment/api",
			Namespace:     "default",
			PrimaryReason: "Unavailable replicas",
			Severity:      "warning",
			Summary:       []string{"0 of 2 replicas available."},
		},
		[]analyzer.AnalysisResult{
			{Resource: "replicaset/api-123", Severity: "warning"},
		},
		[]analyzer.AnalysisResult{
			{Resource: "pod/api-good", Severity: "healthy"},
			{Resource: "pod/api-bad", Severity: "critical"},
		},
		[]analyzer.AnalysisResult{
			{Resource: "secret/api-token", Severity: "healthy"},
			{Resource: "configmap/api-config", Severity: "critical"},
		},
	)

	assert.Equal(t, "v3", result.SchemaVersion)
	assert.Equal(t, "Critical", result.Status)
	assert.Equal(t, "deployment/api", result.Resource)
	assert.Equal(t, "pod/api-bad", result.Pods[0].Resource)
	assert.Equal(t, "configmap/api-config", result.Dependencies[0].Resource)
	assert.Contains(t, result.Summary, "1 replica sets, 2 pods, 2 dependencies")
}
