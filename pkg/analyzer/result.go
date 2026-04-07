package analyzer

// AnalysisResult is the ultimate output of our diagnosis.
// Every rule must return this exact format. By having a strict structure,
// we ensure the CLI output always follows the same beautiful layout, 
// regardless of whether the error was memory, network, or configuration.
type AnalysisResult struct {
	// Identity: What are we looking at?
	Resource  string // e.g., "pod/my-database"
	Namespace string // e.g., "production"

	// Diagnosis: The TL;DR of what happened
	Status        string // e.g., CrashLoopBackOff
	PrimaryReason string // e.g., OOMKilled
	Severity      string // Used for coloring string: "critical" (red), "warning" (yellow), "info" (white)

	// Summary: 1-3 bullet points explaining the problem in human-readable terms
	Summary []string

	// Evidence: Specific data points pulled from the cluster that PROVE the diagnosis
	// (e.g. Exit Code: 137, Memory Limit: 512Mi)
	Evidence []Evidence

	// Actionable next steps
	FixCommands []FixCommand // Exactly what terminal command the user should copy/paste to fix it
	NextChecks  []string     // Commands to run to investigate further

	// Raw Context
	RecentEvents []string
	RecentLogs   string

	// RunbookHint: Optional internal link (e.g. "wiki.company.com/runbooks/oom")
	RunbookHint string
}

// Evidence represents a key/value pair shown in the terminal.
type Evidence struct {
	Label string
	Value string
	Bar   *ProgressBar // Optional UI element to draw an ASCII bar chart (great for Memory/CPU limits)
}

type ProgressBar struct {
	Current int64
	Max     int64
	Unit    string
}

// FixCommand is a combination of what the fix does, and the actual copy/pasteable text.
type FixCommand struct {
	Description string // e.g. "Increase memory limit to 1Gi"
	Command     string // e.g. "kubectl set resources deployment database -c postgres --limits=memory=1Gi"
}
