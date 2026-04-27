package render_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestText_ExplainAndSecondaryFindings(t *testing.T) {
	result := analyzer.AnalysisResult{
		SchemaVersion: "v2",
		Resource:      "pod/api",
		Namespace:     "default",
		Status:        "OOMKilled",
		PrimaryReason: "Container exceeded its memory limit",
		Severity:      "critical",
		Summary:       []string{"The container used more memory than its configured limit."},
		Evidence: []analyzer.Evidence{
			{Label: "Container", Value: "api"},
		},
		Findings: []analyzer.Finding{
			{
				Category:       "Pod",
				ReasonCode:     "OOM_KILLED",
				Confidence:     "high",
				AffectedObject: "container/api",
				Message:        "Container exceeded its memory limit.",
			},
			{
				Category:       "Storage",
				ReasonCode:     "PVC_NOT_FOUND",
				Confidence:     "high",
				AffectedObject: "pod/api",
				Message:        "PVC api-config was not found.",
				Evidence: []analyzer.Evidence{
					{Label: "Error", Value: "pvc api-config not found"},
				},
			},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.Text(result, render.Options{
			Explain:       true,
			ShowSecondary: true,
		}))
	})

	assert.Contains(t, output, "Secondary Findings")
	assert.Contains(t, output, "PVC api-config was not found. (PVC_NOT_FOUND)")
	assert.Contains(t, output, "Explain")
	assert.Contains(t, output, "Primary Diagnosis: OOM_KILLED")
	assert.Contains(t, output, "Confidence: high")
	assert.Contains(t, output, "Affected Object: container/api")
}

func TestText_HidesSecondaryFindingsByDefault(t *testing.T) {
	result := analyzer.AnalysisResult{
		SchemaVersion: "v2",
		Resource:      "pod/api",
		Namespace:     "default",
		Status:        "Warning",
		PrimaryReason: "Primary",
		Severity:      "warning",
		Summary:       []string{"Primary finding."},
		Findings: []analyzer.Finding{
			{ReasonCode: "PRIMARY", Confidence: "medium", Message: "Primary finding."},
			{ReasonCode: "SECONDARY", Confidence: "low", Message: "Secondary finding."},
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, render.Text(result, render.Options{}))
	})

	assert.NotContains(t, output, "Secondary Findings")
	assert.NotContains(t, output, "Secondary finding.")
	assert.NotContains(t, output, "Explain")
	assert.Contains(t, output, "Confidence")
	assert.Contains(t, output, "medium (PRIMARY)")
	assert.NotContains(t, output, "low (SECONDARY)")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer

	fn()

	require.NoError(t, writer.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return buf.String()
}
