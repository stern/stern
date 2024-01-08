package stern

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"testing"
	"text/template"
	"time"

	"github.com/fatih/color"
	"k8s.io/client-go/kubernetes/fake"
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

func TestUpdateTimezoneAndFormat(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name     string
		format   string
		message  string
		expected string
		err      string
	}{
		{
			"normal case",
			"", // default format is used if empty
			"2021-04-18T03:54:44.764981564Z",
			"2021-04-18T12:54:44.764981564+09:00",
			"",
		},
		{
			"padding",
			"",
			"2021-04-18T03:54:44.764981500Z",
			"2021-04-18T12:54:44.764981500+09:00",
			"",
		},
		{
			"timestamp required on non timestamp message",
			"",
			"",
			"",
			"missing timestamp",
		},
		{
			"not UTC",
			"",
			"2021-08-03T01:26:29.953994922+02:00",
			"2021-08-03T08:26:29.953994922+09:00",
			"",
		},
		{
			"RFC3339Nano format removed trailing zeros",
			"",
			"2021-06-20T08:20:30.331385Z",
			"2021-06-20T17:20:30.331385000+09:00",
			"",
		},
		{
			"Specified the short format",
			TimestampFormatShort,
			"2021-06-20T08:20:30.331385Z",
			"06-20 17:20:30",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tailOptions := &TailOptions{
				Location:        location,
				TimestampFormat: tt.format,
			}

			message, err := tailOptions.UpdateTimezoneAndFormat(tt.message)
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
			tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", tmpl, out, io.Discard, &TailOptions{})
			tail.resumeRequest = tt.resumeReq
			if err := tail.ConsumeRequest(context.TODO(), &responseWrapperMock{data: bytes.NewBufferString(logLines)}); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected %s, but actual %s", i, tt.expected, out)
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
		tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", nil, io.Discard, errOut, tt.options)
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
		tail := NewTail(clientset.CoreV1(), "my-node", "my-namespace", "my-pod", "my-container", nil, io.Discard, errOut, tt.options)
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

func TestHighlighIncludedString(t *testing.T) {
	tests := []struct {
		msg      string
		include  []*regexp.Regexp
		expected string
	}{
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			"\x1b[31;1mtest\x1b[0m matched",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			"test not-matched",
		},
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmatched\x1b[0m",
		},
		{
			"test multiple matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmultiple\x1b[0m \x1b[31;1mmatched\x1b[0m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}

func TestIncludeAndHighlightMatchedString(t *testing.T) {
	tests := []struct {
		msg       string
		include   []*regexp.Regexp
		highlight []*regexp.Regexp
		expected  string
	}{
		{
			"test matched with highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`highlight`),
			},
			"\x1b[31;1mtest\x1b[0m matched with \x1b[31;1mhighlight\x1b[0m",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`highlight`),
			},
			"test not-matched",
		},
		{
			"test matched with highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`no-with-highlight`),
				regexp.MustCompile(`with highlight`),
			},
			"test \x1b[31;1mmatched\x1b[0m \x1b[31;1mwith highlight\x1b[0m",
		},
		{
			"test multiple matched with many highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`many`),
				regexp.MustCompile(`highlight`),
			},
			"test \x1b[31;1mmultiple\x1b[0m \x1b[31;1mmatched\x1b[0m with \x1b[31;1mmany\x1b[0m \x1b[31;1mhighlight\x1b[0m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include, Highlight: tt.highlight}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}

func TestHighlightMatchedString(t *testing.T) {
	tests := []struct {
		msg       string
		highlight []*regexp.Regexp
		expected  string
	}{
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			"\x1b[31;1mtest\x1b[0m matched",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			"test not-matched",
		},
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmatched\x1b[0m",
		},
		{
			"test multiple matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmultiple\x1b[0m \x1b[31;1mmatched\x1b[0m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Highlight: tt.highlight}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}
