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
	"time"

	"k8s.io/client-go/kubernetes/fake"
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

func TestUpdateTimezoneIfNeeded(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name        string
		tailOptions *TailOptions
		message     string
		expected    string
		err         string
	}{
		{
			"normal case",
			&TailOptions{
				Timestamps: true,
				Location:   location,
			},
			"2021-04-18T03:54:44.764981564Z Connection: keep-alive",
			"2021-04-18T12:54:44.764981564+09:00 Connection: keep-alive",
			"",
		},
		{
			"padding",
			&TailOptions{
				Timestamps: true,
				Location:   location,
			},
			"2021-04-18T03:54:44.764981500Z Connection: keep-alive",
			"2021-04-18T12:54:44.764981500+09:00 Connection: keep-alive",
			"",
		},
		{
			"no timestamp",
			&TailOptions{
				Timestamps: false,
				Location:   location,
			},
			"Connection: keep-alive",
			"Connection: keep-alive",
			"",
		},
		{
			"timestamp required on non timestamp message",
			&TailOptions{
				Timestamps: true,
				Location:   location,
			},
			"Connection: keep-alive",
			"Connection: keep-alive",
			"missing timestamp",
		},
		{
			"not UTC",
			&TailOptions{
				Timestamps: true,
				Location:   location,
			},
			"2021-08-03T01:26:29.953994922+02:00 Connection: keep-alive",
			"2021-08-03T08:26:29.953994922+09:00 Connection: keep-alive",
			"",
		},
		{
			"RFC3339Nano format removed trailing zeros",
			&TailOptions{
				Timestamps: true,
				Location:   location,
			},
			"2021-06-20T08:20:30.331385Z Connection: keep-alive",
			"2021-06-20T17:20:30.331385000+09:00 Connection: keep-alive",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := tt.tailOptions.UpdateTimezoneIfNeeded(tt.message)
			if tt.expected != message {
				t.Errorf("expected %q, but actual %q", tt.expected, message)
			}

			if err != nil && tt.err != err.Error() {
				t.Errorf("expected %q, but actual %q", tt.err, err)
			}
		})
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

	clientset := fake.NewSimpleClientset()
	for i, tt := range tests {
		out := new(bytes.Buffer)
		tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", tt.tmpl, out, ioutil.Discard, &TailOptions{})

		if err := tail.ConsumeRequest(context.TODO(), tt.request); err != nil {
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
