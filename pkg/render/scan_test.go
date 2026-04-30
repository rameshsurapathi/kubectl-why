package render_test

import (
	"testing"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanText_HidesHealthyByDefault(t *testing.T) {
	result := analyzer.ScanResult{
		SchemaVersion: "v3",
		Namespace:     "default",
		Status:        "Critical",
		Summary:       []string{"1 critical issue", "0 warnings", "1 healthy resource"},
		Results: []analyzer.AnalysisResult{
			{
				Resource:      "pod/bad",
				PrimaryReason: "IMAGE_PULL_FAILED",
				Severity:      "critical",
				Summary:       []string{"Kubernetes cannot pull the image."},
				Findings: []analyzer.Finding{
					{ReasonCode: "IMAGE_PULL_FAILED", Confidence: "high"},
				},
			},
			{
				Resource:      "pod/good",
				PrimaryReason: "Healthy",
				Severity:      "healthy",
				Summary:       []string{"All containers are running and ready."},
			},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.ScanText(result, false))
	})

	assert.Contains(t, output, "namespace/default")
	assert.Contains(t, output, "pod/bad")
	assert.Contains(t, output, "confidence high")
	assert.NotContains(t, output, "pod/good")
}

func TestScanText_ShowsHealthyWhenRequested(t *testing.T) {
	result := analyzer.ScanResult{
		SchemaVersion: "v3",
		Namespace:     "default",
		Status:        "Healthy",
		Summary:       []string{"0 critical issues", "0 warnings", "1 healthy resource"},
		Results: []analyzer.AnalysisResult{
			{
				Resource:      "pod/good",
				PrimaryReason: "Healthy",
				Severity:      "healthy",
				Summary:       []string{"All containers are running and ready."},
			},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.ScanText(result, true))
	})

	assert.Contains(t, output, "pod/good")
}
