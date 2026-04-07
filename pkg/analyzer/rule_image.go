package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// ImagePullRule handles all failures related to fetching the container image.
type ImagePullRule struct{}

func (r *ImagePullRule) Name() string { return "ImagePullBackOff" }

// Match checks all containers for waiting reasons indicating image fetch failure.
func (r *ImagePullRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "ImagePullBackOff" || c.State.WaitingReason == "ErrImagePull") {
			return true
		}
	}
	return false
}

// Analyze pulls the specific image string from the API and attaches it to the evidence.
func (r *ImagePullRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	reason := "ImagePullBackOff"
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && c.State.WaitingReason == "ErrImagePull" {
			reason = "ErrImagePull"
			break
		}
	}

	status := "ImagePullBackOff"
	primaryReason := "Cannot pull container image from registry"
	summaryText := []string{
		"Kubernetes cannot pull the image.",
		"This is usually a wrong tag, deleted image, or auth failure.",
	}
	runbookHint := "kubernetes/image-pull"

	if reason == "ErrImagePull" {
		status = "ErrImagePull"
		primaryReason = "Image pull failed"
		summaryText = []string{
			"First attempt to pull the image failed.",
			"Kubernetes will retry (becoming ImagePullBackOff).",
		}
		runbookHint = "" 
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        status,
		PrimaryReason: primaryReason,
		Severity:      "critical",
		Summary:       summaryText,
		Evidence: []Evidence{
			{Label: "Image", Value: getImageName(signals)},
			{Label: "Error", Value: getImagePullMessage(signals)},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check image name and tag",
				Command: fmt.Sprintf(
					"kubectl describe pod %s -n %s | grep Image", signals.PodName, signals.Namespace),
			},
			{
				Description: "Check imagePullSecrets",
				Command: fmt.Sprintf(
					"kubectl get pod %s -n %s -o jsonpath='{.spec.imagePullSecrets}'", signals.PodName, signals.Namespace),
			},
		},
		NextChecks: []string{
			"Check if the image tag is spelled correctly",
			"Check if imagePullSecrets are valid and linked to the service account",
			"Verify network access from this node to the container registry",
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
		RunbookHint:  runbookHint,
	}
}

func getImageName(signals *kube.PodSignals) string {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting &&
			(c.State.WaitingReason == "ImagePullBackOff" || c.State.WaitingReason == "ErrImagePull") {
			return c.Image
		}
	}
	return "unknown"
}

func getImagePullMessage(signals *kube.PodSignals) string {
	// Check events first — more descriptive
	for _, e := range signals.Events {
		if e.Reason == "Failed" {
			msg := e.Message
			if len(msg) > 75 {
				msg = msg[:72] + "..."
			}
			msg = strings.TrimPrefix(msg, "Error: ")
			return msg
		}
	}
	// Fall back to waiting message
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "ImagePullBackOff" || c.State.WaitingReason == "ErrImagePull") {
			msg := c.State.WaitingMessage
			msg = strings.TrimPrefix(msg, "Error: ")
			return msg
		}
	}
	return "image pull failed"
}
