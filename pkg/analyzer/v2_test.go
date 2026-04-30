package analyzer_test

import (
	"testing"
	"time"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnalyzePod_V2MultipleFindings(t *testing.T) {
	signals := &kube.PodSignals{
		PodName:   "api-7d9",
		Namespace: "default",
		Phase:     "Failed",
		NodeName:  "worker-1",
		Containers: []kube.ContainerSignal{
			{
				Name:         "api",
				RestartCount: 4,
				State: kube.ContainerStateDetail{
					IsTerminated:     true,
					ExitCode:         137,
					TerminatedReason: "OOMKilled",
				},
			},
		},
		Events: []kube.EventSignal{
			{
				Type:    "Warning",
				Reason:  "FailedMount",
				Message: `MountVolume.SetUp failed for volume "config": pvc "api-config" not found`,
			},
		},
	}

	result := analyzer.AnalyzePod(signals)

	assert.Equal(t, "v2", result.SchemaVersion)
	assert.Equal(t, "OOMKilled", result.Status)
	assert.Equal(t, "Container exceeded its memory limit", result.PrimaryReason)
	assert.Len(t, result.Findings, 2)
	assert.Equal(t, "Container exceeded its memory limit", result.Findings[0].ReasonCode)
	assert.Equal(t, "PVC_NOT_FOUND", result.Findings[1].ReasonCode)
}

func TestAnalyzeDeployment_HealthyAndFailingPod(t *testing.T) {
	healthy := analyzer.AnalyzeDeployment(&kube.DeploymentSignals{
		DeploymentName:    "api",
		Namespace:         "default",
		DesiredReplicas:   3,
		ReadyReplicas:     3,
		AvailableReplicas: 3,
		AllHealthy:        true,
	})

	assert.Equal(t, "v2", healthy.SchemaVersion)
	assert.Equal(t, "Healthy", healthy.Status)
	assert.Equal(t, "healthy", healthy.Severity)
	assert.NotEmpty(t, healthy.Findings)

	failing := analyzer.AnalyzeDeployment(&kube.DeploymentSignals{
		DeploymentName: "api",
		Namespace:      "default",
		TotalPods:      2,
		HealthyPods:    1,
		FailingPods:    1,
		FailingPodName: "api-bad",
		FailingPodSignals: &kube.PodSignals{
			PodName:   "api-bad",
			Namespace: "default",
			Phase:     "Pending",
			Events: []kube.EventSignal{
				{
					Type:    "Warning",
					Reason:  "FailedScheduling",
					Message: "0/2 nodes are available: 2 Insufficient memory.",
				},
			},
		},
	})

	assert.Equal(t, "deployment/api", failing.Resource)
	assert.Equal(t, "Pending — cannot be scheduled", failing.Status)
	assert.Contains(t, failing.Evidence[0].Value, "1 healthy")
	assert.Equal(t, "INSUFFICIENT_MEMORY", failing.Findings[0].ReasonCode)
}

func TestAnalyzeRollout_EmbedsPodRootCause(t *testing.T) {
	result := analyzer.AnalyzeDeploymentRollout(&kube.DeploymentSignals{
		DeploymentName:    "api",
		Namespace:         "default",
		DesiredReplicas:   2,
		UpdatedReplicas:   2,
		AvailableReplicas: 1,
		Conditions: []appsv1.DeploymentCondition{
			{
				Type:    appsv1.DeploymentProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  "ProgressDeadlineExceeded",
				Message: "ReplicaSet has timed out progressing.",
			},
		},
		FailingPodName: "api-bad",
		FailingPodSignals: &kube.PodSignals{
			PodName:   "api-bad",
			Namespace: "default",
			Phase:     "Pending",
			InitContainers: []kube.ContainerSignal{
				{
					Name: "init-db",
					State: kube.ContainerStateDetail{
						IsWaiting:     true,
						WaitingReason: "CrashLoopBackOff",
					},
					IsInit: true,
				},
			},
		},
	})

	assert.Equal(t, "v2", result.SchemaVersion)
	assert.Equal(t, "InitContainerCrashLoopBackOff", result.PrimaryReason)
	assert.Len(t, result.Findings, 3)
	assert.Equal(t, "PROGRESS_DEADLINE_EXCEEDED", result.Findings[0].ReasonCode)
	assert.Equal(t, "INIT_CONTAINER_FAILED", result.Findings[1].ReasonCode)
	assert.Equal(t, "Container is repeatedly crashing", result.Findings[2].ReasonCode)
}

func TestAnalyzeJob_FailedIncludesPodFinding(t *testing.T) {
	result := analyzer.AnalyzeJob(&kube.JobSignals{
		JobName:       "batch",
		Namespace:     "default",
		IsFailed:      true,
		Retries:       4,
		BackoffLimit:  3,
		FailedPodName: "batch-xyz",
		FailedPodSignals: &kube.PodSignals{
			PodName:   "batch-xyz",
			Namespace: "default",
			Phase:     "Failed",
			Containers: []kube.ContainerSignal{
				{
					Name: "worker",
					State: kube.ContainerStateDetail{
						IsTerminated: true,
						ExitCode:     2,
					},
				},
			},
		},
	})

	assert.Equal(t, "v2", result.SchemaVersion)
	assert.Equal(t, "Failed", result.Status)
	assert.Equal(t, "Exit Code 2", result.PrimaryReason)
	assert.NotEmpty(t, result.Findings)
	assert.Equal(t, "CONTAINER_EXIT_NONZERO", result.Findings[0].ReasonCode)
	assert.Contains(t, result.FixCommands[0].Command, "kubectl logs batch-xyz")
}

func TestAnalyzeService_States(t *testing.T) {
	noPods := analyzer.AnalyzeService(&kube.ServiceSignals{
		Name:      "api",
		Namespace: "default",
		Type:      "ClusterIP",
		Selector:  map[string]string{"app": "api"},
	})

	assert.Equal(t, "v2", noPods.SchemaVersion)
	assert.Equal(t, "NO_MATCHING_PODS", noPods.PrimaryReason)
	assert.Equal(t, "critical", noPods.Severity)
	assert.Equal(t, "NO_MATCHING_PODS", noPods.Findings[0].ReasonCode)

	healthy := analyzer.AnalyzeService(&kube.ServiceSignals{
		Name:      "api",
		Namespace: "default",
		Type:      "ClusterIP",
		Selector:  map[string]string{"app": "api"},
		MatchingPods: []corev1.Pod{
			{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					},
				},
			},
		},
	})

	assert.Equal(t, "Healthy", healthy.Status)
	assert.Equal(t, "SERVICE_HEALTHY", healthy.PrimaryReason)
	assert.Equal(t, "healthy", healthy.Severity)

	manualEndpoints := analyzer.AnalyzeService(&kube.ServiceSignals{
		Name:      "kubernetes",
		Namespace: "default",
		Type:      "ClusterIP",
		EndpointSlices: []discoveryv1.EndpointSlice{
			{
				Endpoints: []discoveryv1.Endpoint{
					{},
				},
			},
		},
	})

	assert.Equal(t, "Healthy", manualEndpoints.Status)
	assert.Equal(t, "MANUAL_ENDPOINTS", manualEndpoints.PrimaryReason)
	assert.Equal(t, "healthy", manualEndpoints.Severity)
}

func TestAnalyzePVC_ProvisioningAndWaitForFirstConsumer(t *testing.T) {
	provisioning := analyzer.AnalyzePVC(&kube.PVCSignals{
		Name:             "data",
		Namespace:        "default",
		Phase:            "Pending",
		StorageClassName: "fast",
		Events: []kube.EventSignal{
			{
				Reason:  "ProvisioningFailed",
				Message: "failed to provision volume with StorageClass fast",
			},
		},
	})

	assert.Equal(t, "v2", provisioning.SchemaVersion)
	assert.Equal(t, "Dynamic Provisioning Failed", provisioning.PrimaryReason)
	assert.Equal(t, "critical", provisioning.Severity)
	assert.Equal(t, "ProvisioningFailed", provisioning.Findings[0].ReasonCode)

	waiting := analyzer.AnalyzePVC(&kube.PVCSignals{
		Name:             "data",
		Namespace:        "default",
		Phase:            "Pending",
		StorageClassName: "delayed",
		Events: []kube.EventSignal{
			{
				Reason:  "WaitForFirstConsumer",
				Message: "waiting for first consumer to be created before binding",
			},
		},
	})

	assert.Equal(t, "Waiting for Pod", waiting.PrimaryReason)
	assert.Equal(t, "warning", waiting.Severity)
	assert.Equal(t, "WaitForFirstConsumer", waiting.Findings[0].ReasonCode)
}

func TestAnalyzeCronJob_SuspendedAndFailedJob(t *testing.T) {
	suspended := analyzer.AnalyzeCronJob(&kube.CronJobSignals{
		Name:      "nightly",
		Namespace: "default",
		Schedule:  "0 0 * * *",
		Suspend:   true,
	})

	assert.Equal(t, "v2", suspended.SchemaVersion)
	assert.Equal(t, "CRONJOB_SUSPENDED", suspended.PrimaryReason)
	assert.Equal(t, "warning", suspended.Severity)

	failed := analyzer.AnalyzeCronJob(&kube.CronJobSignals{
		Name:      "nightly",
		Namespace: "default",
		Schedule:  "0 0 * * *",
		RecentJobs: []batchv1.Job{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "nightly-123"},
				Status: batchv1.JobStatus{
					Failed: 1,
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
					},
				},
			},
		},
	})

	assert.Equal(t, "Failed", failed.Status)
	assert.Equal(t, "LAST_JOB_FAILED", failed.PrimaryReason)
	assert.Equal(t, "critical", failed.Severity)
}

func TestAnalyzeNode_MultipleFindings(t *testing.T) {
	result := analyzer.AnalyzeNode(&kube.NodeSignals{
		Name:          "worker-1",
		Unschedulable: true,
		Conditions: []corev1.NodeCondition{
			{
				Type:    corev1.NodeReady,
				Status:  corev1.ConditionFalse,
				Reason:  "KubeletNotReady",
				Message: "container runtime is down",
			},
			{
				Type:    corev1.NodeMemoryPressure,
				Status:  corev1.ConditionTrue,
				Reason:  "KubeletHasInsufficientMemory",
				Message: "node has memory pressure",
			},
		},
		Events: []kube.EventSignal{
			{Type: "Warning", Reason: "FreeDiskSpaceFailed", Message: "failed to garbage collect images"},
		},
	})

	assert.Equal(t, "v2", result.SchemaVersion)
	assert.Equal(t, "Degraded", result.Status)
	assert.Equal(t, "critical", result.Severity)
	assert.Len(t, result.Findings, 4)
	assert.Equal(t, "NODE_NOT_READY", result.Findings[0].ReasonCode)
	assert.Equal(t, "NODE_MemoryPressure", result.Findings[1].ReasonCode)
	assert.Equal(t, "NODE_UNSCHEDULABLE", result.Findings[2].ReasonCode)
	assert.Equal(t, "NODE_WARNING_EVENT", result.Findings[3].ReasonCode)
}

func TestAnalyzeCronJob_NoSuccessfulJobsRecently(t *testing.T) {
	lastSchedule := metav1.NewTime(time.Now().Add(-25 * time.Hour))

	result := analyzer.AnalyzeCronJob(&kube.CronJobSignals{
		Name:             "nightly",
		Namespace:        "default",
		Schedule:         "0 0 * * *",
		LastScheduleTime: &lastSchedule,
	})

	assert.Equal(t, "NO_SUCCESSFUL_JOBS_RECENTLY", result.PrimaryReason)
	assert.Equal(t, "warning", result.Severity)
}
