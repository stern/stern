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
	"io"
	"regexp"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// Config contains the config for stern
type Config struct {
	Namespaces            []string
	PodQuery              *regexp.Regexp
	ExcludePodQuery       []*regexp.Regexp
	Timestamps            bool
	TimestampFormat       string
	Location              *time.Location
	ContainerQuery        *regexp.Regexp
	ExcludeContainerQuery []*regexp.Regexp
	ContainerStates       []ContainerState
	Exclude               []*regexp.Regexp
	Include               []*regexp.Regexp
	Highlight             []*regexp.Regexp
	InitContainers        bool
	EphemeralContainers   bool
	Since                 time.Duration
	AllNamespaces         bool
	LabelSelector         labels.Selector
	FieldSelector         fields.Selector
	TailLines             *int64
	Template              *template.Template
	Follow                bool
	Resource              string
	OnlyLogLines          bool
	MaxLogRequests        int
	Stdin                 bool
	DiffContainer         bool
	SamePrefixColor       bool

	Out    io.Writer
	ErrOut io.Writer
}
