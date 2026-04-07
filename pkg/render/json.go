package render

import (
	"encoding/json"
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
)

// JSON takes the final AnalysisResult and prints it out as a structured JSON object.
// This is incredibly useful if someone wants to pipe the output of your tool 
// into jq or another script for automation (e.g. kubectl-why pod my-pod -o json | jq '.PrimaryReason').
func JSON(result analyzer.AnalysisResult) error {
	// MarshalIndent converts the Go struct into a JSON string, with 2 spaces for indentation.
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
