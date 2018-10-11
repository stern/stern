package stern

import "testing"

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
