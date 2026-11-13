package stern

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"
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
		&corev1.Service{
			ObjectMeta: genMeta("svc1"),
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "svc-label",
				},
			},
		},
		// apps
		&appsv1.DaemonSet{
			ObjectMeta: genMeta("ds1"),
			Spec: appsv1.DaemonSetSpec{
				Template: genPodTemplateSpec("ds-label"),
			},
		},
		&appsv1.Deployment{
			ObjectMeta: genMeta("deploy1"),
			Spec: appsv1.DeploymentSpec{
				Template: genPodTemplateSpec("deploy-label"),
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
			desc:  "replicationcontrollers",
			kinds: []string{"rc", "replicationcontrollers", "replicationcontroller"},
			name:  "rc1",
			label: "rc-label",
		},
		{
			desc:  "services",
			kinds: []string{"svc", "services", "service"},
			name:  "svc1",
			label: "svc-label",
		},
		// apps
		{
			desc:  "daemonsets",
			kinds: []string{"ds", "daemonsets", "daemonset"},
			name:  "ds1",
			label: "ds-label",
		},
		{
			desc:  "deployments",
			kinds: []string{"deploy", "deployments", "deployment"},
			name:  "deploy1",
			label: "deploy-label",
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

// logServer serves the log requests of tailTarget. Each request gets the
// response of the next element of responses, and the test fails if there are
// more requests than responses.
type logServer struct {
	t         *testing.T
	mu        sync.Mutex
	requests  []*corev1.PodLogOptions
	responses []func(req *http.Request) io.ReadCloser
}

func (s *logServer) getLogs(_ string, opts *corev1.PodLogOptions) *rest.Request {
	client := &fakerest.RESTClient{Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		s.mu.Lock()
		n := len(s.requests)
		s.requests = append(s.requests, opts)
		s.mu.Unlock()
		if n >= len(s.responses) {
			s.t.Errorf("unexpected log request %d", n+1)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: s.responses[n](req)}, nil
	})}
	return client.Request()
}

func (s *logServer) numRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

type logsCoreV1 struct {
	corev1client.CoreV1Interface
	server *logServer
}

func (c logsCoreV1) Pods(namespace string) corev1client.PodInterface {
	return logsPods{PodInterface: c.CoreV1Interface.Pods(namespace), server: c.server}
}

type logsPods struct {
	corev1client.PodInterface
	server *logServer
}

func (p logsPods) GetLogs(name string, opts *corev1.PodLogOptions) *rest.Request {
	return p.server.getLogs(name, opts)
}

// logs returns a response with the given lines that ends right away.
func logs(lines string, then func()) func(*http.Request) io.ReadCloser {
	return func(*http.Request) io.ReadCloser {
		if then != nil {
			then()
		}
		return io.NopCloser(strings.NewReader(lines))
	}
}

// openLogs returns a response that stays open until the request is canceled.
func openLogs(lines string, then func()) func(*http.Request) io.ReadCloser {
	return func(req *http.Request) io.ReadCloser {
		r, w := io.Pipe()
		go func() {
			_, _ = io.WriteString(w, lines)
			if then != nil {
				then()
			}
			<-req.Context().Done()
			w.CloseWithError(req.Context().Err())
		}()
		return r
	}
}

func TestTailTargetAfterEndedStream(t *testing.T) {
	gracePeriod, pollInterval := endedStreamGracePeriod, targetPollInterval
	endedStreamGracePeriod, targetPollInterval = 100*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { endedStreamGracePeriod, targetPollInterval = gracePeriod, pollInterval })

	genPod := func(containerID string, state corev1.ContainerState) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns", UID: "uid"},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:        "c",
					ContainerID: containerID,
					State:       state,
				}},
			},
		}
	}
	running := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	terminated := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ContainerID: "containerd://1"}}

	tests := []struct {
		name string
		// responses gets a function that applies a pod watch event
		responses func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser
		expected  string
		// sinceTimes are the SinceTime of the log requests after the first one
		sinceTimes []string
	}{
		{
			name: "container terminates",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n", func() { watch(genPod("containerd://1", terminated)) }),
				}
			},
			expected: "a\n",
		},
		{
			name: "watch reports the termination after the stream ends",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n", func() {
						time.AfterFunc(20*time.Millisecond, func() { watch(genPod("containerd://1", terminated)) })
					}),
				}
			},
			expected: "a\n",
		},
		{
			name: "container is restarted",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n", func() { watch(genPod("containerd://2", running)) }),
				}
			},
			expected: "a\n",
		},
		{
			name: "pod is deleted",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n", func() { watch(nil) }),
				}
			},
			expected: "a\n",
		},
		{
			name: "stream ends while the container is running",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n2026-01-01T00:00:01Z b\n", nil),
					logs("2026-01-01T00:00:01Z b\n2026-01-01T00:00:01Z c\n2026-01-01T00:00:02Z d\n", func() {
						watch(genPod("containerd://1", terminated))
					}),
				}
			},
			expected:   "a\nb\nc\nd\n",
			sinceTimes: []string{"2026-01-01T00:00:01Z"},
		},
		{
			name: "resumed stream belongs to a restarted container",
			responses: func(watch func(*corev1.Pod)) []func(*http.Request) io.ReadCloser {
				return []func(*http.Request) io.ReadCloser{
					logs("2026-01-01T00:00:00Z a\n", nil),
					openLogs("2026-01-01T00:00:01Z b\n", func() { watch(genPod("containerd://2", running)) }),
				}
			},
			expected:   "a\nb\n",
			sinceTimes: []string{"2026-01-01T00:00:00Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newTargetFilter(targetFilterConfig{
				podFilter:       regexp.MustCompile(""),
				containerFilter: regexp.MustCompile(".*"),
				containerStates: []ContainerState{ALL_STATES},
			})
			var target *Target
			filter.visit(genPod("containerd://1", running), func(t *Target, _ bool) { target = t })
			if target == nil {
				t.Fatal("the running container was not added")
			}
			watch := func(pod *corev1.Pod) {
				if pod == nil {
					filter.forget("uid")
					return
				}
				filter.visit(pod, func(*Target, bool) {})
			}

			server := &logServer{t: t, responses: tt.responses(watch)}
			client := logsCoreV1{CoreV1Interface: fake.NewSimpleClientset().CoreV1(), server: server}
			tmpl := template.Must(template.New("").Parse("{{.Message}}\n"))
			out := new(bytes.Buffer)
			newTail := func(t *Target) *Tail {
				return NewTail(client, t.Pod, t.Container, tmpl, out, io.Discard, &TailOptions{Follow: true, Location: time.UTC}, false)
			}

			done := make(chan struct{})
			go func() {
				tailTarget(context.Background(), target, filter, newTail, io.Discard)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("tailTarget did not return")
			}

			if n := server.numRequests(); n != len(tt.responses(watch)) {
				t.Errorf("expected %d log requests, but got %d", len(tt.responses(watch)), n)
			}
			for i, sinceTime := range tt.sinceTimes {
				if i+1 >= len(server.requests) {
					break
				}
				opts := server.requests[i+1]
				if opts.SinceTime == nil || opts.SinceTime.UTC().Format(time.RFC3339) != sinceTime {
					t.Errorf("expected request %d to resume since %s, but got %v", i+2, sinceTime, opts.SinceTime)
				}
			}
			if out.String() != tt.expected {
				t.Errorf("expected output %q, but got %q", tt.expected, out.String())
			}
		})
	}
}
