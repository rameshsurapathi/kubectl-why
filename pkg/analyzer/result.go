package analyzer

// AnalysisResult is the ultimate output of our diagnosis.
// Every rule must return this exact format. By having a strict structure,
// we ensure the CLI output always follows the same beautiful layout, 
// regardless of whether the error was memory, network, or configuration.
type AnalysisResult struct {
	// SchemaVersion for JSON backward compatibility (e.g., "v2")
	SchemaVersion string `json:"schemaVersion,omitempty"`

	// Identity: What are we looking at?
	Resource  string `json:"resource"`  // e.g., "pod/my-database"
	Namespace string `json:"namespace"` // e.g., "production"

	// Diagnosis: The TL;DR of what happened
	Status        string `json:"status"`        // e.g., CrashLoopBackOff
	PrimaryReason string `json:"primaryReason"` // e.g., OOMKilled
	Severity      string `json:"severity"`      // Used for coloring string: "critical" (red), "warning" (yellow), "info" (white)

	// Summary: 1-3 bullet points explaining the problem in human-readable terms
	Summary []string `json:"summary"`

	// Evidence: Specific data points pulled from the cluster that PROVE the diagnosis
	// (e.g. Exit Code: 137, Memory Limit: 512Mi)
	Evidence []Evidence `json:"evidence"`

	// Actionable next steps
	FixCommands []FixCommand `json:"fixCommands"` // Exactly what terminal command the user should copy/paste to fix it
	NextChecks  []string     `json:"nextChecks"`  // Commands to run to investigate further

	// Raw Context
	RecentEvents []string `json:"recentEvents"`
	RecentLogs   string   `json:"recentLogs"`

	// RunbookHint: Optional internal link (e.g. "wiki.company.com/runbooks/oom")
	RunbookHint string `json:"runbookHint"`

	// V2: Findings for deeper, multi-issue diagnosis
	Findings []Finding `json:"findings,omitempty"`

	// V3: Better reasoning output
	Reasoning string `json:"reasoning,omitempty"` // Explicitly explain why this rule matched
}

// Finding represents a discrete problem found during diagnosis.
type Finding struct {
	Category       string       `json:"category"`       // e.g., "Pod", "Storage", "Network"
	ReasonCode     string       `json:"reasonCode"`     // e.g., "PROBE_FAILED", "OOM_KILLED"
	Confidence     string       `json:"confidence"`     // "high", "medium", "low"
	AffectedObject string       `json:"affectedObject"` // e.g., "container/nginx"
	Message        string       `json:"message"`
	Reasoning      string       `json:"reasoning,omitempty"` // Explicitly explain why this finding was created
	Evidence       []Evidence   `json:"evidence,omitempty"`
	FixCommands    []FixCommand `json:"fixCommands,omitempty"`
	NextChecks     []string     `json:"nextChecks,omitempty"`
}

// Evidence represents a key/value pair shown in the terminal.
type Evidence struct {
	Label      string       `json:"label"`
	Value      string       `json:"value"`
	Provenance string       `json:"provenance,omitempty"` // JSON Path or source of this evidence (e.g., pod.status.phase)
	Bar        *ProgressBar `json:"bar,omitempty"` // Optional UI element to draw an ASCII bar chart (great for Memory/CPU limits)
}

type ProgressBar struct {
	Current int64  `json:"current"`
	Max     int64  `json:"max"`
	Unit    string `json:"unit"`
}

// FixCommand is a combination of what the fix does, and the actual copy/pasteable text.
type FixCommand struct {
	Description string `json:"description"` // e.g. "Increase memory limit to 1Gi"
	Command     string `json:"command"`     // e.g. "kubectl set resources deployment database -c postgres --limits=memory=1Gi"
	SafetyLevel string `json:"safetyLevel,omitempty"` // "inspect", "low-risk", "mutating", "destructive"
}
