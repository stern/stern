package stern

import (
	"fmt"
	"testing"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func replicaSetPod(name, generateName, hash string) *corev1.Pod {
	var labels map[string]string
	if hash != "" {
		labels = map[string]string{"pod-template-hash": hash}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "default",
			Name:         name,
			GenerateName: generateName,
			Labels:       labels,
		},
	}
}

func TestColorPickerDistinctColorsForSameWorkload(t *testing.T) {
	// Pods of a single ReplicaSet. With the hash-based assignment,
	// "backend-5d9f7f7c6-9qwlz" and "backend-5d9f7f7c6-cbb4t" collide.
	pods := make([]*corev1.Pod, 0, len(colorList))
	for _, suffix := range []string{"2f9nx", "7xk2m", "9qwlz", "cbb4t", "hh6v8", "t9r5d"} {
		pods = append(pods, replicaSetPod("backend-5d9f7f7c6-"+suffix, "backend-5d9f7f7c6-", "5d9f7f7c6"))
	}

	picker := newColorPicker()
	seen := map[int][]string{}
	for _, pod := range pods {
		idx := picker.pick(pod)
		if idx < 0 || idx >= len(colorList) {
			t.Fatalf("color index out of range: %d", idx)
		}
		seen[idx] = append(seen[idx], pod.Name)
	}

	for idx, names := range seen {
		if len(names) > 1 {
			t.Errorf("color %d was assigned to multiple pods of the same ReplicaSet: %v", idx, names)
		}
	}

	// the same pod must keep its color
	for _, pod := range pods {
		if idx := picker.pick(pod); seen[idx][0] != pod.Name {
			t.Errorf("expected pod %s to keep its color, but got the color of %v", pod.Name, seen[idx])
		}
	}
}

func TestColorPickerDistinctColorsForSameDeployment(t *testing.T) {
	// Pods of two ReplicaSets of the same Deployment, as seen during a
	// rolling update. They must be grouped together through the
	// pod-template-hash label so that old and new pods get distinct colors.
	pods := []*corev1.Pod{
		replicaSetPod("backend-5d9f7f7c6-2f9nx", "backend-5d9f7f7c6-", "5d9f7f7c6"),
		replicaSetPod("backend-5d9f7f7c6-7xk2m", "backend-5d9f7f7c6-", "5d9f7f7c6"),
		replicaSetPod("backend-8656df67c4-9qwlz", "backend-8656df67c4-", "8656df67c4"),
		replicaSetPod("backend-8656df67c4-cbb4t", "backend-8656df67c4-", "8656df67c4"),
	}

	picker := newColorPicker()
	seen := map[int][]string{}
	for _, pod := range pods {
		idx := picker.pick(pod)
		seen[idx] = append(seen[idx], pod.Name)
	}

	for idx, names := range seen {
		if len(names) > 1 {
			t.Errorf("color %d was assigned to multiple pods of the same Deployment: %v", idx, names)
		}
	}
}

func TestColorPickerReusesLeastRecentlyUsedColor(t *testing.T) {
	picker := newColorPicker()

	first := picker.pick(replicaSetPod("web-0", "web-", ""))
	for i := 1; i < len(colorList); i++ {
		picker.pick(replicaSetPod(fmt.Sprintf("web-%d", i), "web-", ""))
	}

	// All colors are in use now, so the color assigned least recently
	// (the color of web-0) is reused.
	if got := picker.pick(replicaSetPod(fmt.Sprintf("web-%d", len(colorList)), "web-", "")); got != first {
		t.Errorf("expected the least recently used color %d to be reused, but got %d", first, got)
	}
}

func TestColorPickerKeepsHashBasedColorForUngroupedPods(t *testing.T) {
	picker := newColorPicker()
	for _, name := range []string{"stern", "my-pod", "etcd-control-plane"} {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name}}
		if got, want := picker.pick(pod), int(colorIndex(name)); got != want {
			t.Errorf("expected pod %s to keep the hash-based color %d, but got %d", name, want, got)
		}
	}
}

func TestColorGroupKey(t *testing.T) {
	tests := []struct {
		desc string
		pod  *corev1.Pod
		want string
	}{
		{
			desc: "pod of a Deployment (pod-template-hash is removed)",
			pod:  replicaSetPod("backend-5d9f7f7c6-2f9nx", "backend-5d9f7f7c6-", "5d9f7f7c6"),
			want: "default/backend-",
		},
		{
			desc: "pod of a bare ReplicaSet",
			pod:  replicaSetPod("backend-2f9nx", "backend-", ""),
			want: "default/backend-",
		},
		{
			desc: "pod of a StatefulSet",
			pod:  replicaSetPod("web-0", "web-", ""),
			want: "default/web-",
		},
		{
			desc: "pod without generateName",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "etcd-control-plane"},
			},
			want: "kube-system/etcd-control-plane",
		},
		{
			desc: "pod-template-hash not matching generateName is kept",
			pod:  replicaSetPod("backend-2f9nx", "backend-", "5d9f7f7c6"),
			want: "default/backend-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := colorGroupKey(tt.pod); got != tt.want {
				t.Errorf("expected %q, but got %q", tt.want, got)
			}
		})
	}
}

func TestParseColors(t *testing.T) {
	tests := []struct {
		desc            string
		podColors       []string
		containerColors []string
		want            [][2]*color.Color
		wantError       bool
	}{
		{
			desc:            "both pod and container colors are specified",
			podColors:       []string{"91", "92", "93"},
			containerColors: []string{"31", "32", "33"},
			want: [][2]*color.Color{
				{color.New(color.FgHiRed), color.New(color.FgRed)},
				{color.New(color.FgHiGreen), color.New(color.FgGreen)},
				{color.New(color.FgHiYellow), color.New(color.FgYellow)},
			},
		},
		{
			desc:            "only pod colors are specified",
			podColors:       []string{"91", "92", "93"},
			containerColors: []string{},
			want: [][2]*color.Color{
				{color.New(color.FgHiRed), color.New(color.FgHiRed)},
				{color.New(color.FgHiGreen), color.New(color.FgHiGreen)},
				{color.New(color.FgHiYellow), color.New(color.FgHiYellow)},
			},
		},
		{
			desc:            "multiple attributes",
			podColors:       []string{"4;91"},
			containerColors: []string{"38;2;255;97;136"},
			want: [][2]*color.Color{
				{
					color.New(color.Underline, color.FgHiRed),
					color.New(38, 2, 255, 97, 136), // 24-bit color
				},
			},
		},
		{
			desc:            "spaces are ignored",
			podColors:       []string{"  91 ", "\t92\t"},
			containerColors: []string{},
			want: [][2]*color.Color{
				{color.New(color.FgHiRed), color.New(color.FgHiRed)},
				{color.New(color.FgHiGreen), color.New(color.FgHiGreen)},
			},
		},
		// error patterns
		{
			desc:            "only container colors are specified",
			podColors:       []string{},
			containerColors: []string{"31", "32", "33"},
			wantError:       true,
		},
		{
			desc:            "both pod and container colors are empty",
			podColors:       []string{},
			containerColors: []string{},
			wantError:       true,
		},
		{
			desc:            "invalid color",
			podColors:       []string{"a"},
			containerColors: []string{""},
			wantError:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			colorList, err := parseColors(tt.podColors, tt.containerColors)

			if tt.wantError {
				if err == nil {
					t.Error("expected err, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if len(tt.want) != len(colorList) {
				t.Fatalf("expected colorList of size %d, but got %d", len(tt.want), len(colorList))
			}

			for i, wantPair := range tt.want {
				gotPair := colorList[i]
				if !wantPair[0].Equals(gotPair[0]) {
					t.Errorf("colorList[%d][0]: expected %v, but got %v", i, wantPair[0], gotPair[0])
				}
				if !wantPair[1].Equals(gotPair[1]) {
					t.Errorf("colorList[%d][1]: expected %v, but got %v", i, wantPair[1], gotPair[1])
				}
			}
		})
	}
}
