package stern

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListMultipleNamespaces(t *testing.T) {
	const numNamespaces, numPods = 20, 200

	var objects []runtime.Object
	var namespaces []string
	for n := 0; n < numNamespaces; n++ {
		ns := fmt.Sprintf("ns%d", n)
		namespaces = append(namespaces, ns)
		for p := 0; p < numPods; p++ {
			objects = append(objects, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("pod%d", p),
					Namespace: ns,
					Labels:    map[string]string{"app.kubernetes.io/instance": fmt.Sprintf("%s-app%d", ns, p)},
				},
			})
		}
	}
	// a pod without the label is ignored
	objects = append(objects, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nolabel", Namespace: "ns0"}})

	client := fake.NewSimpleClientset(objects...)
	labels, err := List(context.Background(), client, &Config{Namespaces: namespaces})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(labels) != numNamespaces*numPods {
		t.Errorf("expected %d label values, but got %d", numNamespaces*numPods, len(labels))
	}
	for n := 0; n < numNamespaces; n++ {
		for p := 0; p < numPods; p++ {
			key := fmt.Sprintf("ns%d-app%d", n, p)
			if labels[key] != "app.kubernetes.io/instance" {
				t.Errorf("expected %q to map to the label key, but got %q", key, labels[key])
			}
		}
	}
}
