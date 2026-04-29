package render_test

import (
	"testing"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceTraceText_HidesHealthyPodsByDefault(t *testing.T) {
	result := analyzer.ServiceTraceResult{
		SchemaVersion:      "v3",
		Resource:           "service/api",
		Namespace:          "default",
		Status:             "Critical",
		EndpointCount:      2,
		ReadyEndpointCount: 1,
		Summary:            []string{"Service matches 2 pods (1 ready, 1 failing)."},
		Service: analyzer.AnalysisResult{
			Resource:      "service/api",
			PrimaryReason: "SOME_PODS_FAILING",
			Severity:      "warning",
			Summary:       []string{"Service matches 2 pods (1 ready, 1 failing)."},
			Findings: []analyzer.Finding{
				{ReasonCode: "SOME_PODS_FAILING", Confidence: "high"},
			},
		},
		BackingPods: []analyzer.AnalysisResult{
			{
				Resource:      "pod/api-bad",
				PrimaryReason: "Image pull failed",
				Severity:      "critical",
				Summary:       []string{"Kubernetes cannot pull the image."},
				Findings: []analyzer.Finding{
					{ReasonCode: "IMAGE_PULL_FAILED", Confidence: "high"},
				},
			},
			{
				Resource:      "pod/api-good",
				PrimaryReason: "Healthy",
				Severity:      "healthy",
			},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.ServiceTraceText(result, false))
	})

	assert.Contains(t, output, "service/api")
	assert.Contains(t, output, "1/2 endpoints ready")
	assert.Contains(t, output, "pod/api-bad")
	assert.Contains(t, output, "IMAGE_PULL_FAILED")
	assert.NotContains(t, output, "pod/api-good")
}

func TestServiceTraceText_ExplainsMissingPodBackends(t *testing.T) {
	result := analyzer.ServiceTraceResult{
		SchemaVersion:      "v3",
		Resource:           "service/kubernetes",
		Namespace:          "default",
		Status:             "Healthy",
		EndpointCount:      1,
		ReadyEndpointCount: 1,
		Service: analyzer.AnalysisResult{
			Resource:      "service/kubernetes",
			PrimaryReason: "MANUAL_ENDPOINTS",
			Severity:      "healthy",
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.ServiceTraceText(result, false))
	})

	assert.Contains(t, output, "No Kubernetes pod backends found")
}

func TestDeploymentTraceText_HidesHealthyComponentsByDefault(t *testing.T) {
	result := analyzer.DeploymentTraceResult{
		SchemaVersion: "v3",
		Resource:      "deployment/api",
		Namespace:     "default",
		Status:        "Critical",
		Deployment: analyzer.AnalysisResult{
			Resource:      "deployment/api",
			PrimaryReason: "Unavailable replicas",
			Severity:      "warning",
			Summary:       []string{"0 of 2 replicas available."},
		},
		ReplicaSets: []analyzer.AnalysisResult{
			{Resource: "replicaset/api-123", PrimaryReason: "Unavailable replicas", Severity: "warning"},
			{Resource: "replicaset/api-old", PrimaryReason: "Healthy", Severity: "healthy"},
		},
		Pods: []analyzer.AnalysisResult{
			{
				Resource:      "pod/api-bad",
				PrimaryReason: "Image pull failed",
				Severity:      "critical",
				Findings: []analyzer.Finding{
					{ReasonCode: "IMAGE_PULL_FAILED", Confidence: "high"},
				},
			},
			{Resource: "pod/api-good", PrimaryReason: "Healthy", Severity: "healthy"},
		},
		Dependencies: []analyzer.AnalysisResult{
			{
				Resource:      "configmap/api-config",
				PrimaryReason: "ConfigMap Not Found",
				Severity:      "critical",
				Findings: []analyzer.Finding{
					{ReasonCode: "CONFIGMAP_NOT_FOUND", Confidence: "high"},
				},
			},
			{Resource: "secret/api-token", PrimaryReason: "Dependency exists", Severity: "healthy"},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.DeploymentTraceText(result, false))
	})

	assert.Contains(t, output, "deployment/api")
	assert.Contains(t, output, "ReplicaSets")
	assert.Contains(t, output, "replicaset/api-123")
	assert.NotContains(t, output, "replicaset/api-old")
	assert.Contains(t, output, "pod/api-bad")
	assert.NotContains(t, output, "pod/api-good")
	assert.Contains(t, output, "configmap/api-config")
	assert.NotContains(t, output, "secret/api-token")
}
