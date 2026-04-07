package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// Rule defines a single diagnostic check
type Rule interface {
	Name() string
	Match(signals *kube.PodSignals) bool
	Analyze(signals *kube.PodSignals) AnalysisResult
}

// registry is a private, global slice that holds your arsenal of rules.
// Order matters! A pod might have multiple things wrong with it, but the
// rule that registers *first* gets the priority to report the error.
var registry []Rule

func init() {
	// Explicit priority ordering to ensure root causes shadow generic symptoms
	registry = []Rule{
		&EvictedRule{},
		&OOMKilledRule{},
		&ImagePullRule{},
		&ConfigErrorRule{},
		&CannotRunRule{},
		&SegfaultRule{},
		&AppCrashRule{},
		&CrashLoopRule{},
		&PendingRule{},
	}
}

// RegisterRule allows dynamic addition of rules
func RegisterRule(r Rule) {
	registry = append(registry, r)
}

// extractEventStrings is a helper for rules to format event history
func extractEventStrings(events []kube.EventSignal, max int) []string {
	var out []string
	count := 0
	for _, e := range events {
		if count >= max {
			break
		}
		// Skip trivial normal events if we just want warnings or errors
		out = append(out, fmt.Sprintf("%s: %s", e.Reason, e.Message))
		count++
	}
	return out
}
