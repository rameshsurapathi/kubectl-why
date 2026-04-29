package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCollectPodDependencies_SkipsOptionalConfigAndSecretRefs(t *testing.T) {
	optional := true
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			ServiceAccountName: "worker",
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "pull-secret"},
			},
			Volumes: []corev1.Volume{
				{
					Name: "kube-api-access-abcde",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ConfigMap: &corev1.ConfigMapProjection{
										LocalObjectReference: corev1.LocalObjectReference{Name: "kube-root-ca.crt"},
									},
								},
							},
						},
					},
				},
				{
					Name: "required-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "required-config"},
						},
					},
				},
				{
					Name: "optional-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "optional-config"},
							Optional:             &optional,
						},
					},
				},
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "data",
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name: "app",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "required-secret"},
							},
						},
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "optional-env-config"},
								Optional:             &optional,
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "optional-key-secret"},
									Optional:             &optional,
								},
							},
						},
					},
				},
			},
		},
	}

	pvcs := map[string]bool{}
	configMaps := map[string]bool{}
	secrets := map[string]bool{}
	serviceAccounts := map[string]bool{}

	collectPodDependencies(pod, pvcs, configMaps, secrets, serviceAccounts)

	assert.True(t, pvcs["data"])
	assert.True(t, configMaps["required-config"])
	assert.False(t, configMaps["kube-root-ca.crt"])
	assert.False(t, configMaps["optional-config"])
	assert.False(t, configMaps["optional-env-config"])
	assert.True(t, secrets["required-secret"])
	assert.True(t, secrets["pull-secret"])
	assert.False(t, secrets["optional-key-secret"])
	assert.True(t, serviceAccounts["worker"])
}

func TestMissingDependencyIncludesFinding(t *testing.T) {
	result := missingDependency("configmap", "app-config", "default")

	assert.Equal(t, "configmap/app-config", result.Resource)
	assert.Equal(t, "critical", result.Severity)
	assert.Equal(t, "CONFIGMAP_NOT_FOUND", result.Findings[0].ReasonCode)
	assert.Equal(t, "high", result.Findings[0].Confidence)
}
