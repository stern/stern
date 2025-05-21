package stern

import (
	"bytes"
	"context"
	"fmt"
	"github.com/fatih/color"
	"io"
	"reflect"
	"regexp"
	"testing"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDetermineColor(t *testing.T) {
	podName := "stern"
	containerName := "foo"
	diffContainer := false
	podColor1, containerColor1 := determineColor(podName, containerName, diffContainer)
	podColor2, containerColor2 := determineColor(podName, containerName, diffContainer)

	if podColor1 != podColor2 {
		t.Errorf("expected color for pod to be the same between invocations but was %v and %v",
			podColor1, podColor2)
	}
	if containerColor1 != containerColor2 {
		t.Errorf("expected color for container to be the same between invocations but was %v and %v",
			containerColor1, containerColor2)
	}
}

func TestDetermineColorDiffContainer(t *testing.T) {
	podName := "stern"
	containerName1 := "foo"
	containerName2 := "bar"
	diffContainer := true
	podColor1, containerColor1 := determineColor(podName, containerName1, diffContainer)
	podColor2, containerColor2 := determineColor(podName, containerName2, diffContainer)

	if podColor1 != podColor2 {
		t.Errorf("expected color for pod to be the same between invocations but was %v and %v",
			podColor1, podColor2)
	}
	if containerColor1 == containerColor2 {
		t.Errorf("expected color for container to be different between invocations but was the same: %v",
			containerColor1)
	}
}

func TestConsumeStreamTail(t *testing.T) {
	logLines := `2023-02-13T21:20:30.000000001Z line 1
2023-02-13T21:20:30.000000002Z line 2
2023-02-13T21:20:31.000000001Z line 3
2023-02-13T21:20:31.000000002Z line 4`
	tmpl := template.Must(template.New("").Parse(`{{printf "%s (%s/%s/%s/%s)\n" .Message .NodeName .Namespace .PodName .ContainerName}}`))

	tests := []struct {
		name      string
		resumeReq *ResumeRequest
		expected  []byte
	}{
		{
			name: "normal",
			expected: []byte(`line 1 (my-node/my-namespace/my-pod/my-container)
line 2 (my-node/my-namespace/my-pod/my-container)
line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
		{
			name:      "ResumeRequest LinesToSkip=1",
			resumeReq: &ResumeRequest{Timestamp: "2023-02-13T21:20:30Z", LinesToSkip: 1},
			expected: []byte(`line 2 (my-node/my-namespace/my-pod/my-container)
line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
		{
			name:      "ResumeRequest LinesToSkip=2",
			resumeReq: &ResumeRequest{Timestamp: "2023-02-13T21:20:30Z", LinesToSkip: 2},
			expected: []byte(`line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
		{
			name:      "ResumeRequest LinesToSkip=3 (exceed)",
			resumeReq: &ResumeRequest{Timestamp: "2023-02-13T21:20:30Z", LinesToSkip: 3},
			expected: []byte(`line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
		{
			name:      "ResumeRequest does not match",
			resumeReq: &ResumeRequest{Timestamp: "2222-22-22T21:20:30Z", LinesToSkip: 3},
			expected: []byte(`line 1 (my-node/my-namespace/my-pod/my-container)
line 2 (my-node/my-namespace/my-pod/my-container)
line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
	}

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					NodeName: "my-node",
				},
			}
			tail := NewTail(clientset.CoreV1(), pod, "my-container", tmpl, out, io.Discard, &TailOptions{}, false)
			tail.resumeRequest = tt.resumeReq
			if err := tail.ConsumeRequest(context.TODO(), &responseWrapperMock{data: bytes.NewBufferString(logLines)}); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected `%s`, but actual `%s`", i, tt.expected, out)
			}
		})
	}
}

func TestHighlight(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()
	coloredLine := colorHighlight("line")

	tmpl := template.Must(template.New("").Parse(`{{printf "%s (%s/%s/%s/%s)\n" .Message .NodeName .Namespace .PodName .ContainerName}}`))

	tests := []struct {
		name     string
		logLine  string
		expected []byte
	}{
		{
			name: "normal",
			logLine: `2023-02-13T21:20:30.000000001Z line 1
2023-02-13T21:20:30.000000002Z line 2
2023-02-13T21:20:31.000000001Z line 3
2023-02-13T21:20:31.000000002Z line 4`,
			expected: []byte(fmt.Sprintf(`%s 1 (my-node/my-namespace/my-pod/my-container)
%s 2 (my-node/my-namespace/my-pod/my-container)
%s 3 (my-node/my-namespace/my-pod/my-container)
%s 4 (my-node/my-namespace/my-pod/my-container)
`, coloredLine, coloredLine, coloredLine, coloredLine)),
		},
		{
			name: "no highlight",
			logLine: `2023-02-13T21:20:30.000000001Z log 1
2023-02-13T21:20:30.000000002Z log 2
2023-02-13T21:20:31.000000001Z log 3
2023-02-13T21:20:31.000000002Z log 4`,
			expected: []byte(`log 1 (my-node/my-namespace/my-pod/my-container)
log 2 (my-node/my-namespace/my-pod/my-container)
log 3 (my-node/my-namespace/my-pod/my-container)
log 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
	}

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", tmpl, out, io.Discard, &TailOptions{Highlight: []*regexp.Regexp{regexp.MustCompile("line")}}, false)
			if err := tail.ConsumeRequest(context.TODO(), &responseWrapperMock{data: bytes.NewBufferString(tt.logLine)}); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected `%s`, but actual `%s`", i, tt.expected, out)
			}
		})
	}
}

func TestInclude(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	coloredLine := colorHighlight("line")

	tmpl := template.Must(template.New("").Parse(`{{printf "%s (%s/%s/%s/%s)\n" .Message .NodeName .Namespace .PodName .ContainerName}}`))

	tests := []struct {
		name     string
		logLine  string
		expected []byte
	}{
		{
			name: "normal",
			logLine: `2023-02-13T21:20:30.000000001Z line 1
2023-02-13T21:20:30.000000002Z line 2
2023-02-13T21:20:31.000000001Z line 3
2023-02-13T21:20:31.000000002Z line 4`,
			expected: []byte(fmt.Sprintf(`%s 1 (my-node/my-namespace/my-pod/my-container)
%s 2 (my-node/my-namespace/my-pod/my-container)
%s 3 (my-node/my-namespace/my-pod/my-container)
%s 4 (my-node/my-namespace/my-pod/my-container)
`, coloredLine, coloredLine, coloredLine, coloredLine)),
		},
		{
			name: "full excluded",
			logLine: `2023-02-13T21:20:30.000000001Z log 1
2023-02-13T21:20:30.000000002Z log 2
2023-02-13T21:20:31.000000001Z log 3
2023-02-13T21:20:31.000000002Z log 4`,
			expected: []byte(""),
		},

		{
			name: "one included",
			logLine: `2023-02-13T21:20:30.000000001Z log 1
2023-02-13T21:20:30.000000002Z line 2
2023-02-13T21:20:31.000000001Z log 3
2023-02-13T21:20:31.000000002Z log 4`,
			expected: []byte(fmt.Sprintf(`%s 2 (my-node/my-namespace/my-pod/my-container)
`, coloredLine)),
		},
	}

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", tmpl, out, io.Discard, &TailOptions{Include: []*regexp.Regexp{regexp.MustCompile("line")}}, false)
			if err := tail.ConsumeRequest(context.TODO(), &responseWrapperMock{data: bytes.NewBufferString(tt.logLine)}); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected `%s`, but actual `%s`", i, tt.expected, out)
			}
		})
	}
}

type responseWrapperMock struct {
	data io.Reader
}

func (r *responseWrapperMock) DoRaw(context.Context) ([]byte, error) {
	data, _ := io.ReadAll(r.data)
	return data, nil
}

func (r *responseWrapperMock) Stream(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(r.data), nil
}

func TestPrintStarting(t *testing.T) {
	tests := []struct {
		options  *TailOptions
		expected []byte
	}{
		{
			&TailOptions{},
			[]byte("+ my-pod › my-container\n"),
		},
		{
			&TailOptions{
				Namespace: true,
			},
			[]byte("+ my-namespace my-pod › my-container\n"),
		},
		{
			&TailOptions{
				OnlyLogLines: true,
			},
			[]byte{},
		},
		{
			&TailOptions{
				Namespace:    true,
				OnlyLogLines: true,
			},
			[]byte{},
		},
	}

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		errOut := new(bytes.Buffer)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-pod",
			},
		}
		tail := NewTail(clientset.CoreV1(), pod, "my-container", nil, io.Discard, errOut, tt.options, false)
		tail.printStarting()

		if !bytes.Equal(tt.expected, errOut.Bytes()) {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, errOut)
		}
	}
}

func TestPrintStopping(t *testing.T) {
	tests := []struct {
		options  *TailOptions
		expected []byte
	}{
		{
			&TailOptions{},
			[]byte("- my-pod › my-container\n"),
		},
		{
			&TailOptions{
				Namespace: true,
			},
			[]byte("- my-namespace my-pod › my-container\n"),
		},
		{
			&TailOptions{
				OnlyLogLines: true,
			},
			[]byte{},
		},
		{
			&TailOptions{
				Namespace:    true,
				OnlyLogLines: true,
			},
			[]byte{},
		},
	}

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		errOut := new(bytes.Buffer)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-pod",
			},
		}
		tail := NewTail(clientset.CoreV1(), pod, "my-container", nil, io.Discard, errOut, tt.options, false)
		tail.printStopping()

		if !bytes.Equal(tt.expected, errOut.Bytes()) {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, errOut)
		}
	}
}

func TestResumeRequestShouldSkip(t *testing.T) {
	tests := []struct {
		rr         ResumeRequest
		timestamps []string
		expected   []bool
	}{
		{
			rr:         ResumeRequest{Timestamp: "t1", LinesToSkip: 1},
			timestamps: []string{"t1", "t1"},
			expected:   []bool{true, false},
		},
		{
			rr:         ResumeRequest{Timestamp: "t1", LinesToSkip: 3},
			timestamps: []string{"t1", "t1", "t1", "t1"},
			expected:   []bool{true, true, true, false},
		},
		{
			rr:         ResumeRequest{Timestamp: "t1", LinesToSkip: 3},
			timestamps: []string{"t2", "t2"},
			expected:   []bool{false, false},
		},
	}
	for _, tt := range tests {
		var actual []bool
		for _, ts := range tt.timestamps {
			actual = append(actual, tt.rr.shouldSkip(ts))
		}
		if !reflect.DeepEqual(tt.expected, actual) {
			t.Errorf("expected %v, but actual %v", tt.expected, actual)
		}
	}
}

func TestRemoveSubsecond(t *testing.T) {
	tests := []struct {
		ts       string
		expected string
	}{
		{
			ts:       "2023-02-14T05:36:39.902767599Z",
			expected: "2023-02-14T05:36:39Z",
		},
		{
			ts:       "2023-02-14T05:36:39.1Z",
			expected: "2023-02-14T05:36:39Z",
		},
		{
			ts:       "2023-02-14T05:36:39Z",
			expected: "2023-02-14T05:36:39Z",
		},
		{
			ts:       "1.1",
			expected: "1",
		},
		{
			ts:       "10.1",
			expected: "10",
		},
		{
			ts:       "",
			expected: "",
		},
		{
			ts:       ".",
			expected: ".",
		},
		{
			ts:       ".1",
			expected: "",
		},
	}
	for _, tt := range tests {
		actual := removeSubsecond(tt.ts)
		if tt.expected != actual {
			t.Errorf("expected %v, but actual %v", tt.expected, actual)
		}
	}
}
