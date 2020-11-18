package stern

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"testing"
	"text/template"

	"k8s.io/client-go/rest"
)

func TestDetermineColor(t *testing.T) {
	podName := "stern"
	podColor1, containerColor1 := determineColor(podName)
	podColor2, containerColor2 := determineColor(podName)

	if podColor1 != podColor2 {
		t.Errorf("expected color for pod to be the same between invocations but was %v and %v",
			podColor1, podColor2)
	}
	if containerColor1 != containerColor2 {
		t.Errorf("expected color for container to be the same between invocations but was %v and %v",
			containerColor1, containerColor2)
	}
}

func TestIsIncludeTestOptions(t *testing.T) {
	msg := "this is a log message"

	tests := []struct {
		include  []*regexp.Regexp
		expected bool
	}{
		{
			include:  []*regexp.Regexp{},
			expected: true,
		},
		{
			include: []*regexp.Regexp{
				regexp.MustCompile(`this is not`),
			},
			expected: false,
		},
		{
			include: []*regexp.Regexp{
				regexp.MustCompile(`this is`),
			},
			expected: true,
		},
	}

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include}
		if o.IsInclude(msg) != tt.expected {
			t.Errorf("%d: expected %s, but actual %s", i, fmt.Sprint(tt.expected), fmt.Sprint(!tt.expected))
		}
	}
}

func TestConsumeStreamTail(t *testing.T) {
	tests := []struct {
		tmpl     *template.Template
		request  rest.ResponseWrapper
		expected []byte
	}{
		{
			tmpl: template.Must(template.New("").Parse(`{{printf "%s (%s/%s/%s/%s)\n" .Message .NodeName .Namespace .PodName .ContainerName}}`)),
			request: &responseWrapperMock{
				data: bytes.NewBufferString(`line 1
line 2
line 3
line 4`),
			},
			expected: []byte(`line 1 (my-node/my-namespace/my-pod/my-container)
line 2 (my-node/my-namespace/my-pod/my-container)
line 3 (my-node/my-namespace/my-pod/my-container)
line 4 (my-node/my-namespace/my-pod/my-container)
`),
		},
	}

	for i, tt := range tests {
		tail := NewTail("my-node", "my-namespace", "my-pod", "my-container", tt.tmpl, &TailOptions{})
		out := new(bytes.Buffer)

		if err := tail.ConsumeRequest(context.TODO(), tt.request, out); err != nil {
			t.Fatalf("%d: unexpected err %v", i, err)
		}

		if !bytes.Equal(tt.expected, out.Bytes()) {
			t.Errorf("%d: expected %s, but actual %s", i, tt.expected, out)
		}
	}
}

type responseWrapperMock struct {
	data io.Reader
}

func (r *responseWrapperMock) DoRaw(context.Context) ([]byte, error) {
	data, _ := ioutil.ReadAll(r.data)
	return data, nil
}

func (r *responseWrapperMock) Stream(context.Context) (io.ReadCloser, error) {
	return ioutil.NopCloser(r.data), nil
}
