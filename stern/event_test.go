//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package stern

import (
	"bytes"
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func newEvent(kind, name, typ, reason, message string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns1", Name: reason},
		InvolvedObject: corev1.ObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: "ns1",
		},
		Type:    typ,
		Reason:  reason,
		Message: message,
	}
}

func TestFormatEvent(t *testing.T) {
	color.NoColor = true

	tests := []struct {
		desc  string
		event *corev1.Event
		want  string
	}{
		{
			desc:  "normal event",
			event: newEvent("Pod", "pod1", corev1.EventTypeNormal, "Started", "Started container app"),
			want:  "event Normal Pod/pod1 Started: Started container app\n",
		},
		{
			desc:  "warning event trims message",
			event: newEvent("Pod", "pod1", corev1.EventTypeWarning, "BackOff", "  Back-off restarting failed container\n"),
			want:  "event Warning Pod/pod1 BackOff: Back-off restarting failed container\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := formatEvent(tt.event); got != tt.want {
				t.Errorf("formatEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPodEventMatcher(t *testing.T) {
	config := &Config{
		PodQuery:        regexp.MustCompile("^web-"),
		ExcludePodQuery: []*regexp.Regexp{regexp.MustCompile("canary")},
	}
	match := podEventMatcher(config)

	tests := []struct {
		desc  string
		event *corev1.Event
		want  bool
	}{
		{"matching pod", newEvent("Pod", "web-1", "Normal", "Started", "m"), true},
		{"non-pod object", newEvent("Deployment", "web-1", "Normal", "ScalingReplicaSet", "m"), false},
		{"pod not matching query", newEvent("Pod", "db-1", "Normal", "Started", "m"), false},
		{"excluded pod", newEvent("Pod", "web-canary", "Normal", "Started", "m"), false},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := match(tt.event); got != tt.want {
				t.Errorf("match(%s) = %v, want %v", tt.event.InvolvedObject.Name, got, tt.want)
			}
		})
	}
}

// syncBuffer is a bytes.Buffer safe for concurrent writes and reads.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestConsumeEvents(t *testing.T) {
	color.NoColor = true

	fw := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := &syncBuffer{}
	// only accept pods named "pod1"
	match := func(e *corev1.Event) bool {
		return e.InvolvedObject.Kind == "Pod" && e.InvolvedObject.Name == "pod1"
	}

	done := make(chan struct{})
	go func() {
		_ = consumeEvents(ctx, fw.ResultChan(), match, buf)
		close(done)
	}()

	fw.Add(newEvent("Pod", "pod2", corev1.EventTypeNormal, "Started", "ignored"))
	fw.Add(newEvent("Pod", "pod1", corev1.EventTypeWarning, "BackOff", "back-off"))
	// Modified events should be skipped.
	fw.Modify(newEvent("Pod", "pod1", corev1.EventTypeNormal, "Pulled", "modified"))

	want := "event Warning Pod/pod1 BackOff: back-off\n"
	deadline := time.After(5 * time.Second)
	for buf.String() != want {
		select {
		case <-deadline:
			t.Fatalf("consumeEvents wrote %q, want %q", buf.String(), want)
		case <-time.After(10 * time.Millisecond):
		}
	}

	fw.Stop()
	<-done
}
