package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// promptHandler invokes the interactive prompt and updates config.LabelSelector with the selected value.
func promptHandler(ctx context.Context, client kubernetes.Interface, config *stern.Config, out io.Writer) error {
	labelsMap, err := stern.List(ctx, client, config)
	if err != nil {
		return err
	}

	if len(labelsMap) == 0 {
		return errors.New("No matching labels")
	}

	var choices []string

	for key := range labelsMap {
		choices = append(choices, key)
	}

	sort.Strings(choices)

	choice, err := selectPods(choices)
	if err != nil {
		return err
	}

	selector := fmt.Sprintf("%v=%v", labelsMap[choice], choice)

	fmt.Fprintf(out, "Selector: %v\n", color.BlueString(selector))

	labelSelector, err := labels.Parse(selector)
	if err != nil {
		return err
	}

	config.LabelSelector = labelSelector

	return nil
}

// selectPods surfaces an interactive prompt for selecting an app.kubernetes.io/instance.
func selectPods(pods []string) (string, error) {
	arrow := survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "❯"
		icons.SelectFocus.Text = "❯"
		icons.Question.Format = "blue"
		icons.SelectFocus.Format = "blue"
	})

	prompt := &survey.Select{
		Message: "Select \"app.kubernetes.io/instance\" label value:",
		Options: pods,
	}

	var pod string

	if err := survey.AskOne(prompt, &pod, arrow); err != nil {
		return "", err
	}

	return pod, nil
}
