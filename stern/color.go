package stern

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var colorList = [][2]*color.Color{
	{color.New(color.FgHiCyan), color.New(color.FgCyan)},
	{color.New(color.FgHiGreen), color.New(color.FgGreen)},
	{color.New(color.FgHiMagenta), color.New(color.FgMagenta)},
	{color.New(color.FgHiYellow), color.New(color.FgYellow)},
	{color.New(color.FgHiBlue), color.New(color.FgBlue)},
	{color.New(color.FgHiRed), color.New(color.FgRed)},
}

// colorPicker assigns colors to pods so that pods created by the same
// workload (e.g. replicas of a Deployment or a StatefulSet) get distinct
// colors as long as unused colors remain in colorList.
type colorPicker struct {
	mu       sync.Mutex
	seq      int64
	assigned map[string]int     // namespace/pod name -> assigned color index
	groups   map[string][]int64 // group key -> sequence when each color was last assigned
}

func newColorPicker() *colorPicker {
	return &colorPicker{
		assigned: make(map[string]int),
		groups:   make(map[string][]int64),
	}
}

// podColorPicker assigns pod colors for the process.
var podColorPicker = newColorPicker()

// pick returns the color index for the pod. The same pod always gets the
// same color index within the process. A new pod starts at the hash-based
// index, so pods that do not share a group with other pods keep the same
// color as previous stern versions, and then the color assigned least
// recently within its group is chosen so that pods created by the same
// workload get distinct colors (anti-affinity).
func (p *colorPicker) pick(pod *corev1.Pod) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	podKey := pod.Namespace + "/" + pod.Name
	if idx, ok := p.assigned[podKey]; ok {
		return idx
	}

	groupKey := colorGroupKey(pod)
	lastUsed, ok := p.groups[groupKey]
	if !ok || len(lastUsed) != len(colorList) {
		lastUsed = make([]int64, len(colorList))
		p.groups[groupKey] = lastUsed
	}

	start := int(colorIndex(pod.Name))
	best := start
	for i := 1; i < len(lastUsed); i++ {
		c := (start + i) % len(lastUsed)
		if lastUsed[c] < lastUsed[best] {
			best = c
		}
	}

	p.seq++
	lastUsed[best] = p.seq
	p.assigned[podKey] = best
	return best
}

// colorGroupKey returns the key used to group pods for the color
// anti-affinity. Pods created by the same workload (ReplicaSet,
// StatefulSet, DaemonSet, Job, etc.) share the same generateName. The
// pod-template-hash is removed from the generateName so that pods from
// different ReplicaSets of the same Deployment are also grouped together.
func colorGroupKey(pod *corev1.Pod) string {
	group := pod.GenerateName
	if hash, ok := pod.Labels[appsv1.DefaultDeploymentUniqueLabelKey]; ok {
		group = strings.TrimSuffix(group, hash+"-")
	}
	if group == "" {
		group = pod.Name
	}
	return pod.Namespace + "/" + group
}

func SetColorList(podColors, containerColors []string) error {
	colors, err := parseColors(podColors, containerColors)
	if err != nil {
		return err
	}
	colorList = colors
	return nil
}

func parseColors(podColors, containerColors []string) ([][2]*color.Color, error) {
	if len(podColors) == 0 {
		return nil, errors.New("pod-colors must not be empty")
	}
	if len(containerColors) == 0 {
		// if containerColors is empty, use podColors as containerColors
		return createColorPairs(podColors, podColors)
	}
	if len(containerColors) != len(podColors) {
		return nil, errors.New("pod-colors and container-colors must have the same length")
	}
	return createColorPairs(podColors, containerColors)
}

func createColorPairs(podColors, containerColors []string) ([][2]*color.Color, error) {
	colorList := make([][2]*color.Color, 0, len(podColors))
	for i := 0; i < len(podColors); i++ {
		podColor, err := sgrSequenceToColor(podColors[i])
		if err != nil {
			return nil, err
		}
		containerColor, err := sgrSequenceToColor(containerColors[i])
		if err != nil {
			return nil, err
		}
		colorList = append(colorList, [2]*color.Color{podColor, containerColor})
	}
	return colorList, nil
}

// sgrSequenceToColor converts a string representing SGR sequence
// separated by ";" into a *color.Color instance.
// For example, "31;4" means red foreground with underline.
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func sgrSequenceToColor(s string) (*color.Color, error) {
	parts := strings.Split(s, ";")
	attrs := make([]color.Attribute, 0, len(parts))
	for _, part := range parts {
		attr, err := strconv.ParseInt(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, color.Attribute(attr))
	}
	return color.New(attrs...), nil
}
