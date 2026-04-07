package analyzer_test

import (
    "encoding/json"
    "os"
    "testing"

    corev1 "k8s.io/api/core/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
    "github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// loadPodFixture reads a saved kubectl get pod -o json
// file and converts it to PodSignals for testing.
// Optional synthetic events array can be embedded.
func loadPodFixture(t *testing.T, filename string) *kube.PodSignals {
    t.Helper()

    data, err := os.ReadFile("testdata/" + filename)
    require.NoError(t, err,
        "fixture file %q not found — "+
            "run: kubectl get pod <name> -o json > "+
            "pkg/analyzer/testdata/%s", filename, filename)

    var pod corev1.Pod
    err = json.Unmarshal(data, &pod)
    require.NoError(t, err,
        "failed to parse fixture %q", filename)

    var supplement struct {
        Events []kube.EventSignal `json:"events"`
    }
    _ = json.Unmarshal(data, &supplement)

    signals := podToSignals(&pod)
    if len(supplement.Events) > 0 {
        signals.Events = supplement.Events
    }

    return signals
}

// podToSignals converts a corev1.Pod into PodSignals
// without making any API calls — for testing only
func podToSignals(pod *corev1.Pod) *kube.PodSignals {
    signals := &kube.PodSignals{
        PodName:   pod.Name,
        Namespace: pod.Namespace,
        Phase:     string(pod.Status.Phase),
        PodReason: pod.Status.Reason,
        NodeName:  pod.Spec.NodeName,
    }

    for _, c := range pod.Status.Conditions {
        signals.Conditions = append(
            signals.Conditions,
            kube.ConditionSignal{
                Type:    string(c.Type),
                Status:  string(c.Status),
                Reason:  c.Reason,
                Message: c.Message,
            })
    }

    for _, cs := range pod.Status.ContainerStatuses {
        signals.Containers = append(
            signals.Containers,
            kube.BuildContainerSignal(cs, false))
    }

    for _, cs := range pod.Status.InitContainerStatuses {
        signals.InitContainers = append(
            signals.InitContainers,
            kube.BuildContainerSignal(cs, true))
    }

    return signals
}

// ── Tests ─────────────────────────────────────────────────

func TestAnalyze_ImagePullBackOff(t *testing.T) {
    signals := loadPodFixture(t, "imagepullbackoff_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "ImagePullBackOff", result.Status)
    assert.Equal(t, "critical", result.Severity)
    assert.NotEmpty(t, result.FixCommands,
        "should provide fix commands")

    // Should mention the image somewhere in evidence
    found := false
    for _, e := range result.Evidence {
        if e.Label == "Image" {
            found = true
            assert.Contains(t, e.Value, "doesntexist")
        }
    }
    assert.True(t, found,
        "evidence should include Image field")
}

func TestAnalyze_CreateContainerConfigError(t *testing.T) {
    signals := loadPodFixture(t,
        "createcontainerconfigerror_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "CreateContainerConfigError",
        result.Status)
    assert.Equal(t, "critical", result.Severity)

    // Should mention configmap in evidence
    found := false
    for _, e := range result.Evidence {
        if e.Label == "Message" {
            found = true
            assert.Contains(t, e.Value, "not found")
        }
    }
    assert.True(t, found,
        "evidence should include Message field")
}

func TestAnalyze_CrashLoopBackOff(t *testing.T) {
    signals := loadPodFixture(t, "crashloop_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "CrashLoopBackOff", result.Status)
    assert.Equal(t, "critical", result.Severity)
    assert.NotEmpty(t, result.FixCommands)
}

func TestAnalyze_OOMKilled(t *testing.T) {
    signals := loadPodFixture(t, "oomkilled_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "OOMKilled", result.Status)
    assert.Equal(t, "critical", result.Severity)

    // Should mention exit code 137
    found := false
    for _, e := range result.Evidence {
        if e.Label == "Exit Code" {
            found = true
            assert.Equal(t, "137", e.Value)
        }
    }
    assert.True(t, found,
        "evidence should include exit code 137")
}

func TestAnalyze_Pending(t *testing.T) {
    signals := loadPodFixture(t, "pending_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "Pending — cannot be scheduled", result.Status)
    assert.Equal(t, "warning", result.Severity)
    assert.NotEmpty(t, result.NextChecks)
}

func TestAnalyze_Healthy(t *testing.T) {
    signals := loadPodFixture(t, "healthy_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "healthy", result.Severity)
    assert.Equal(t, "Running", result.Status)
}

func TestAnalyze_UnknownPod_HasFallback(t *testing.T) {
    // A completely empty signals should not panic
    // and should return something useful
    signals := &kube.PodSignals{
        PodName:   "mystery-pod",
        Namespace: "default",
        Phase:     "Unknown",
    }
    result := analyzer.AnalyzePod(signals)

    assert.NotEmpty(t, result.Status)
    assert.NotEmpty(t, result.PrimaryReason)
    assert.NotPanics(t, func() {
        analyzer.AnalyzePod(signals)
    })
}

func TestAnalyze_AppCrash(t *testing.T) {
    t.Run("fixture_exitcode", func(t *testing.T) {
        signals := loadPodFixture(t, "appcrash_pod.json")
        result := analyzer.AnalyzePod(signals)

        assert.Equal(t, "Exit Code 1", result.Status)
        assert.Equal(t, "critical", result.Severity)

        found := false
        for _, e := range result.Evidence {
            if e.Label == "Exit Code" && e.Value == "1" {
                found = true
            }
        }
        assert.True(t, found, "evidence should include exit code 1")
    })

    t.Run("log_scanning", func(t *testing.T) {
        // Directly construct signals with exit code 1 AND logs containing a panic.
        // This exercises the smart log-scanning path in AppCrashRule.Analyze which
        // was NOT reached by the fixture (fixture has no RecentLogs).
        signals := &kube.PodSignals{
            PodName:   "log-crash-pod",
            Namespace: "default",
            Phase:     "Failed",
            RecentLogs: "Starting app\n" +
                "panic: runtime error: invalid memory address or nil pointer dereference\n" +
                "goroutine 1 [running]:\n" +
                "main.main()\n",
            Containers: []kube.ContainerSignal{
                {
                    Name:  "app",
                    Ready: false,
                    State: kube.ContainerStateDetail{
                        IsTerminated: true,
                        ExitCode:     1,
                    },
                },
            },
        }
        result := analyzer.AnalyzePod(signals)

        assert.Equal(t, "Exit Code 1", result.Status)
        assert.Equal(t, "critical", result.Severity)

        // The smart log-parser should have extracted the panic line as evidence
        logMatchFound := false
        for _, e := range result.Evidence {
            if e.Label == "Log Match 1" {
                logMatchFound = true
                assert.Contains(t, e.Value, "panic:")
            }
        }
        assert.True(t, logMatchFound,
            "AppCrashRule should extract panic lines into Log Match evidence")
    })
}

func TestAnalyze_Segfault(t *testing.T) {
    signals := loadPodFixture(t, "segfault_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "Exit Code 139", result.Status)
    assert.Equal(t, "critical", result.Severity)
}

func TestAnalyze_CannotRun(t *testing.T) {
    signals := loadPodFixture(t, "cannotrun_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "ContainerCannotRun", result.Status)
    assert.Equal(t, "critical", result.Severity)
}

func TestAnalyze_Evicted(t *testing.T) {
    signals := loadPodFixture(t, "evicted_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "Evicted", result.Status)
    assert.Equal(t, "critical", result.Severity)
}
