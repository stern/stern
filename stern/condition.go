package stern

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type Condition struct {
	Name  v1.PodConditionType
	Value v1.ConditionStatus
}

// NewCondition returns a Condition struct for a given conditionString
func NewCondition(conditionString string) (Condition, error) {
	// condition can be: condition-name or condition-name=condition-value
	conditionNameString := strings.ToLower(conditionString)
	conditionValueString := "true"
	if equalsIndex := strings.Index(conditionNameString, "="); equalsIndex != -1 {
		conditionValueString = conditionNameString[equalsIndex+1:]
		conditionNameString = conditionNameString[0:equalsIndex]
	}

	var conditionName v1.PodConditionType

	validConditions := []v1.PodConditionType{
		v1.ContainersReady,
		v1.PodInitialized,
		v1.PodReady,
		v1.PodScheduled,
		v1.DisruptionTarget,
		v1.PodReadyToStartContainers,
	}

	for _, validCondition := range validConditions {
		if strings.ToLower(string(validCondition)) == conditionNameString {
			conditionName = validCondition
			break
		}
	}

	if conditionName == "" {
		validConditionsStrings := make([]string, len(validConditions))
		for i, val := range validConditions {
			validConditionsStrings[i] = string(val)
		}
		return Condition{}, fmt.Errorf("condition should be one of '%s'", strings.Join(validConditionsStrings, "', '"))
	}

	var conditionValue v1.ConditionStatus

	validValues := []v1.ConditionStatus{
		v1.ConditionTrue,
		v1.ConditionFalse,
		v1.ConditionUnknown,
	}

	for _, validValue := range validValues {
		if strings.ToLower(string(validValue)) == conditionValueString {
			conditionValue = validValue
			break
		}
	}

	if conditionValue == "" {
		validValuesStrings := make([]string, len(validValues))
		for i, val := range validValues {
			validValuesStrings[i] = string(val)
		}
		return Condition{}, fmt.Errorf("condition value should be one of '%s'", strings.Join(validValuesStrings, "', '"))
	}

	return Condition{
		Name:  conditionName,
		Value: conditionValue,
	}, nil
}

// Match returns if pod matches the condition
func (conditionConfig Condition) Match(pod *v1.Pod) bool {
	// Ready condition for pods owned by Jobs will be True as soon as the pod
	// starts running, which makes it hard to use stern to watch the progress
	// of a Job. To work around this, consider the pod as not Ready if it is
	// owned by a Job and is running (and consider it Ready once it stops).
	if conditionConfig.Name == v1.PodReady && pod.OwnerReferences != nil {
		for _, ownerReference := range pod.OwnerReferences {
			if ownerReference.Kind == "Job" {
				return true
			}
		}
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionConfig.Name {
			return condition.Status == conditionConfig.Value
		}
	}

	return false
}
