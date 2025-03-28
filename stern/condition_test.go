package stern

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCondition(t *testing.T) {
	tests := []struct {
		conditionString string
		expected        Condition
		isError         bool
	}{
		{
			"Ready",
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionTrue,
			},
			false,
		},
		{
			"ready=true",
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionTrue,
			},
			false,
		},
		{
			"Ready=False",
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionFalse,
			},
			false,
		},
		{
			"ready=Unknown",
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionUnknown,
			},
			false,
		},
		{
			"beautiful",
			Condition{},
			true,
		},
		{
			"ready=NotYet",
			Condition{},
			true,
		},
	}

	for i, tt := range tests {
		condition, err := NewCondition(tt.conditionString)

		if tt.expected != condition {
			t.Errorf("%d: expected %v, but actual %v", i, tt.expected, condition)
		}

		if (tt.isError && err == nil) || (!tt.isError && err != nil) {
			t.Errorf("%d: expected error is %v, but actual %v", i, tt.isError, err)
		}
	}
}

func TestConditionMatch(t *testing.T) {
	tests := []struct {
		condition Condition
		pod       v1.Pod
		expected  bool
	}{
		{
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionTrue,
			},
			v1.Pod{
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			true,
		},
		{
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionTrue,
			},
			v1.Pod{
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodReady,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			false,
		},
		{
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionTrue,
			},
			v1.Pod{
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodInitialized,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			false,
		},
		{
			Condition{
				Name:  v1.PodReady,
				Value: v1.ConditionFalse,
			},
			v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Job",
						},
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			true,
		},
	}

	for i, tt := range tests {
		actual := tt.condition.Match(&tt.pod)

		if tt.expected != actual {
			t.Errorf("%d: expected %v, but actual %v", i, tt.expected, actual)
		}
	}
}
