package analyzer_test

import (
	"testing"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/stretchr/testify/assert"
)

func TestBuildScanResult_RanksUnhealthyResources(t *testing.T) {
	result := analyzer.BuildScanResult("default", []analyzer.AnalysisResult{
		{
			Resource: "pod/healthy",
			Status:   "Running",
			Severity: "healthy",
		},
		{
			Resource:      "service/api",
			Status:        "Degraded",
			PrimaryReason: "NO_READY_PODS",
			Severity:      "critical",
		},
		{
			Resource:      "deployment/api",
			Status:        "Progressing",
			PrimaryReason: "UNAVAILABLE_REPLICAS",
			Severity:      "warning",
		},
	})

	assert.Equal(t, "v3", result.SchemaVersion)
	assert.Equal(t, "Critical", result.Status)
	assert.Equal(t, "service/api", result.Results[0].Resource)
	assert.Equal(t, "deployment/api", result.Results[1].Resource)
	assert.Equal(t, "pod/healthy", result.Results[2].Resource)
	assert.Equal(t, []string{
		"1 critical issue",
		"1 warning",
		"1 healthy resource",
	}, result.Summary)
}

func TestBuildScanResult_HealthyNamespace(t *testing.T) {
	result := analyzer.BuildScanResult("default", []analyzer.AnalysisResult{
		{Resource: "pod/api", Status: "Running", Severity: "healthy"},
	})

	assert.Equal(t, "Healthy", result.Status)
	assert.Equal(t, []string{
		"0 critical issues",
		"0 warnings",
		"1 healthy resource",
	}, result.Summary)
}
