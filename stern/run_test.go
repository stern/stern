package stern

import (
	"bytes"
	"context"
	"regexp"
	"sort"
	"strings"
	"testing"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunPodResourceIsExactMatch(t *testing.T) {
	genPod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:        "c",
					ContainerID: "containerd://" + name,
					State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				}},
			},
		}
	}

	tests := []struct {
		resource string
		expected []string
	}{
		{"pod/web.1", []string{"web.1"}},
		{"po/web.1", []string{"web.1"}},
		{"pod/web", []string{"web"}},
		{"pod/web.+", nil},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			client := fake.NewSimpleClientset(genPod("web"), genPod("web.1"), genPod("webx1"), genPod("web.10"))
			out := new(bytes.Buffer)
			config := &Config{
				Namespaces:      []string{"ns"},
				PodQuery:        regexp.MustCompile(""),
				ContainerQuery:  regexp.MustCompile(".*"),
				ContainerStates: []ContainerState{ALL_STATES},
				Location:        time.UTC,
				LabelSelector:   labels.Everything(),
				FieldSelector:   fields.Everything(),
				Template:        template.Must(template.New("").Parse("{{.PodName}}\n")),
				Resource:        tt.resource,
				MaxLogRequests:  5,
				OnlyLogLines:    true,
				Out:             out,
				ErrOut:          new(bytes.Buffer),
			}
			if err := Run(context.Background(), client, config); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var actual []string
			if s := strings.TrimSpace(out.String()); s != "" {
				actual = strings.Split(s, "\n")
			}
			sort.Strings(actual)
			if strings.Join(actual, ",") != strings.Join(tt.expected, ",") {
				t.Errorf("expected pods %v, but got %v", tt.expected, actual)
			}
		})
	}
}
