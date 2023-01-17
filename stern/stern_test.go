package stern

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRetrieveLabelsFromResource(t *testing.T) {
	genMeta := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name:      name,
			Namespace: "ns1",
		}
	}
	genPodTemplateSpec := func(name string) corev1.PodTemplateSpec {
		return corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": name,
				},
			},
		}
	}
	objs := []runtime.Object{
		// core
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "ns1",
				Labels: map[string]string{
					"app": "pod-label",
				},
			},
		},
		&corev1.ReplicationController{
			ObjectMeta: genMeta("rc1"),
			Spec: corev1.ReplicationControllerSpec{
				Template: &corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "rc-label",
						},
					},
				},
			},
		},
		// apps
		&appsv1.Deployment{
			ObjectMeta: genMeta("deploy1"),
			Spec: appsv1.DeploymentSpec{
				Template: genPodTemplateSpec("deploy-label"),
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: genMeta("ds1"),
			Spec: appsv1.DaemonSetSpec{
				Template: genPodTemplateSpec("ds-label"),
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: genMeta("rs1"),
			Spec: appsv1.ReplicaSetSpec{
				Template: genPodTemplateSpec("rs-label"),
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: genMeta("sts1"),
			Spec: appsv1.StatefulSetSpec{
				Template: genPodTemplateSpec("sts-label"),
			},
		},
		// batch
		&batchv1.Job{
			ObjectMeta: genMeta("job1"),
			Spec: batchv1.JobSpec{
				Template: genPodTemplateSpec("job-label"),
			},
		},
		&batchv1.CronJob{
			ObjectMeta: genMeta("cj1"),
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: genPodTemplateSpec("cj-label"),
					},
				},
			},
		},
	}
	client := fake.NewSimpleClientset(objs...)
	tests := []struct {
		desc      string
		kinds     []string
		name      string
		label     string
		wantError bool
	}{
		// core
		{
			desc:  "pods",
			kinds: []string{"po", "pods", "pod"},
			name:  "pod1",
			label: "pod-label",
		},
		{
			desc:  "replicationcontrollers",
			kinds: []string{"rc", "replicationcontrollers", "replicationcontroller"},
			name:  "rc1",
			label: "rc-label",
		},
		// apps
		{
			desc:  "deployments",
			kinds: []string{"deploy", "deployments", "deployment"},
			name:  "deploy1",
			label: "deploy-label",
		},
		{
			desc:  "daemonsets",
			kinds: []string{"ds", "daemonsets", "daemonset"},
			name:  "ds1",
			label: "ds-label",
		},
		{
			desc:  "replicasets",
			kinds: []string{"rs", "replicasets", "replicaset"},
			name:  "rs1",
			label: "rs-label",
		},
		{
			desc:  "statefulsets",
			kinds: []string{"sts", "statefulsets", "statefulset"},
			name:  "sts1",
			label: "sts-label",
		},
		// batch
		{
			desc:  "jobs",
			kinds: []string{"job", "jobs"},
			name:  "job1",
			label: "job-label",
		},
		{
			desc:  "cronjobs",
			kinds: []string{"cj", "cronjobs", "cronjob"},
			name:  "cj1",
			label: "cj-label",
		},
		// invalid
		{
			desc:      "invalid",
			kinds:     []string{"", "unknown"},
			name:      "dummy",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, kind := range tt.kinds {
				labels, err := retrieveLabelsFromResource(context.Background(), client, "ns1", kind, tt.name)
				if tt.wantError {
					if err == nil {
						t.Errorf("expected error, but got no error")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				expectedLabels := map[string]string{"app": tt.label}
				if !reflect.DeepEqual(expectedLabels, labels) {
					t.Errorf("expected %v, but actual %v", expectedLabels, labels)
				}

				// test not found
				_, err = retrieveLabelsFromResource(context.Background(), client, "ns1", kind, "not-found")
				if !kerrors.IsNotFound(err) {
					t.Errorf("expected not found, but actual %v", err)
				}
			}
		})
	}
}
