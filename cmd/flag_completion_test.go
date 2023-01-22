package cmd

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRetrieveNamesFromResource(t *testing.T) {
	genMeta := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name:      name,
			Namespace: "ns1",
		}
	}
	objs := []runtime.Object{
		&corev1.Pod{ObjectMeta: genMeta("pod1")},
		&corev1.Pod{ObjectMeta: genMeta("pod2")},
		&corev1.Pod{ObjectMeta: genMeta("pod3")},
		&corev1.ReplicationController{ObjectMeta: genMeta("rc1")},
		&corev1.Service{ObjectMeta: genMeta("svc1")},
		&appsv1.Deployment{ObjectMeta: genMeta("deploy1")},
		&appsv1.Deployment{ObjectMeta: genMeta("deploy2")},
		&appsv1.DaemonSet{ObjectMeta: genMeta("ds1")},
		&appsv1.DaemonSet{ObjectMeta: genMeta("ds2")},
		&appsv1.ReplicaSet{ObjectMeta: genMeta("rs1")},
		&appsv1.ReplicaSet{ObjectMeta: genMeta("rs2")},
		&appsv1.StatefulSet{ObjectMeta: genMeta("sts1")},
		&appsv1.StatefulSet{ObjectMeta: genMeta("sts2")},
		&batchv1.Job{ObjectMeta: genMeta("job1")},
		&batchv1.Job{ObjectMeta: genMeta("job2")},
	}
	client := fake.NewSimpleClientset(objs...)
	tests := []struct {
		desc      string
		kinds     []string
		expected  []string
		wantError bool
	}{
		// core
		{
			desc:     "pods",
			kinds:    []string{"po", "pods", "pod"},
			expected: []string{"pod1", "pod2", "pod3"},
		},
		{
			desc:     "replicationcontrollers",
			kinds:    []string{"rc", "replicationcontrollers", "replicationcontroller"},
			expected: []string{"rc1"},
		},
		// apps
		{
			desc:     "deployments",
			kinds:    []string{"deploy", "deployments", "deployment"},
			expected: []string{"deploy1", "deploy2"},
		},
		{
			desc:     "daemonsets",
			kinds:    []string{"ds", "daemonsets", "daemonset"},
			expected: []string{"ds1", "ds2"},
		},
		{
			desc:     "replicasets",
			kinds:    []string{"rs", "replicasets", "replicaset"},
			expected: []string{"rs1", "rs2"},
		},
		{
			desc:     "statefulsets",
			kinds:    []string{"sts", "statefulsets", "statefulset"},
			expected: []string{"sts1", "sts2"},
		},
		// batch
		{
			desc:     "jobs",
			kinds:    []string{"job", "jobs"},
			expected: []string{"job1", "job2"},
		},
		// invalid
		{
			desc:      "invalid",
			kinds:     []string{"", "unknown"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, kind := range tt.kinds {
				names, err := retrieveNamesFromResource(context.Background(), client, "ns1", kind)
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
				if !reflect.DeepEqual(tt.expected, names) {
					t.Errorf("expected %v, but actual %v", tt.expected, names)
				}
				// expect empty slice with no error when no objects are found in the valid resource
				names, err = retrieveNamesFromResource(context.Background(), client, "not-matched", kind)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if len(names) != 0 {
					t.Errorf("expected empty slice, but got %v", names)
					return
				}
			}
		})
	}
}
