// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package equality_test

import (
	"testing"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	"github.com/projectcontour/contour-operator/controller/contour"
	utilequality "github.com/projectcontour/contour-operator/util/equality"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testName  = "test"
	testNs    = testName + "-ns"
	testImage = "test-image:main"
)

func TestJobConfigChanged(t *testing.T) {
	zero := int32(0)

	testCases := []struct {
		description string
		mutate      func(*batchv1.Job)
		expect      bool
	}{
		{
			description: "if nothing changed",
			mutate:      func(_ *batchv1.Job) {},
			expect:      false,
		},
		{
			description: "if the job labels are removed",
			mutate: func(job *batchv1.Job) {
				job.Labels = map[string]string{}
			},
			expect: true,
		},
		{
			description: "if the contour owning label is removed",
			mutate: func(job *batchv1.Job) {
				delete(job.Spec.Template.Labels, operatorv1alpha1.OwningContourLabel)
			},
			expect: true,
		},
		{
			description: "if the container image is changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Template.Spec.Containers[0].Image = "foo:latest"
			},
			expect: true,
		},
		{
			description: "if parallelism is changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Parallelism = &zero
			},
			expect: true,
		},
		// Completions is immutable, so performing an equality comparison is unneeded.
		{
			description: "if backoffLimit is changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.BackoffLimit = &zero
			},
			expect: true,
		},
		{
			description: "if service account name is changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Template.Spec.ServiceAccountName = "foo"
			},
			expect: true,
		},
		{
			description: "if container commands are changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Template.Spec.Containers[0].Command = []string{"foo"}
			},
			expect: true,
		},
		{
			description: "if container env vars are changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Template.Spec.Containers[0].Env[0] = corev1.EnvVar{
					Name: "foo",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				}
			},
			expect: true,
		},
		{
			description: "if security context is changed",
			mutate: func(job *batchv1.Job) {
				job.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
					RunAsUser:    nil,
					RunAsGroup:   nil,
					RunAsNonRoot: nil,
				}
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		ctr := &operatorv1alpha1.Contour{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "test-job-ns",
			},
		}

		expected, err := contour.DesiredJob(ctr, testImage)
		if err != nil {
			t.Errorf("invalid job: %w", err)
		}

		mutated := expected.DeepCopy()
		tc.mutate(mutated)
		if updated, changed := utilequality.JobConfigChanged(mutated, expected); changed != tc.expect {
			t.Errorf("%s, expect jobConfigChanged to be %t, got %t", tc.description, tc.expect, changed)
		} else if changed {
			if _, changedAgain := utilequality.JobConfigChanged(updated, expected); changedAgain {
				t.Errorf("%s, jobConfigChanged does not behave as a fixed point function", tc.description)
			}
		}
	}
}

func TestDeploymentConfigChanged(t *testing.T) {
	testCases := []struct {
		description string
		mutate      func(deployment *appsv1.Deployment)
		expect      bool
	}{
		{
			description: "if nothing changes",
			mutate:      func(_ *appsv1.Deployment) {},
			expect:      false,
		},
		{
			description: "if replicas is changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Replicas = nil
			},
			expect: true,
		},
		{
			description: "if selector is changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Selector = &metav1.LabelSelector{}
			},
			expect: true,
		},
		{
			description: "if the container image is changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Template.Spec.Containers[0].Image = "foo:latest"
			},
			expect: true,
		},
		{
			description: "if a volume is changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Template.Spec.Volumes = []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/foo",
							},
						},
					}}
			},
			expect: true,
		},
		{
			description: "if container commands are changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Template.Spec.Containers[0].Command = []string{"foo"}
			},
			expect: true,
		},
		{
			description: "if container args are changed",
			mutate: func(deploy *appsv1.Deployment) {
				deploy.Spec.Template.Spec.Containers[0].Args = []string{"foo", "bar", "baz"}
			},
			expect: true,
		},
		{
			description: "if probe values are set to default values",
			mutate: func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe.Handler.HTTPGet.Scheme = "HTTP"
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds = int32(1)
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe.PeriodSeconds = int32(10)
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe.SuccessThreshold = int32(1)
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe.FailureThreshold = int32(3)
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.TimeoutSeconds = int32(1)
				// ReadinessProbe InitialDelaySeconds and PeriodSeconds are not set at defaults,
				// so they are omitted.
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.SuccessThreshold = int32(1)
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.FailureThreshold = int32(3)
			},
			expect: false,
		},
	}

	for _, tc := range testCases {
		ctr := &operatorv1alpha1.Contour{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNs,
			},
		}

		original, err := contour.DesiredDeployment(ctr, testImage)
		if err != nil {
			t.Errorf("invalid deployment: %w", err)
		}

		mutated := original.DeepCopy()
		tc.mutate(mutated)
		if updated, changed := utilequality.DeploymentConfigChanged(original, mutated); changed != tc.expect {
			t.Errorf("%s, expect deploymentConfigChanged to be %t, got %t", tc.description, tc.expect, changed)
		} else if changed {
			if _, changedAgain := utilequality.DeploymentConfigChanged(updated, mutated); changedAgain {
				t.Errorf("%s, deploymentConfigChanged does not behave as a fixed point function", tc.description)
			}
		}
	}
}