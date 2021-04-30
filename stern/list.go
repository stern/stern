package stern

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stern/stern/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// List returns a list of all
func List(ctx context.Context, config *Config) (map[string]string, error) {
	clientConfig := kubernetes.NewClientConfig(config.KubeConfig, config.ContextName)
	cc, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := corev1client.NewForConfig(cc)
	if err != nil {
		return nil, err
	}

	var namespaces []string
	// A specific namespace is ignored if all-namespaces is provided.
	if config.AllNamespaces {
		namespaces = []string{""}
	} else {
		namespaces = config.Namespaces
		if len(namespaces) == 0 {
			n, _, err := clientConfig.Namespace()
			if err != nil {
				return nil, errors.Wrap(err, "unable to get default namespace")
			}
			namespaces = []string{n}
		}
	}

	labels := make(map[string]string)
	options := metav1.ListOptions{}

	// Iterate through provided namespaces.
	for _, n := range namespaces {
		pods, err := clientset.Pods(n).List(ctx, options)

		if err != nil {
			return nil, err
		}

		match := "app.kubernetes.io/instance"
		// Iterate through pods in namespace, looking for matching labels.
		for _, pod := range pods.Items {
			key := pod.Labels[match]

			if key == "" {
				continue
			}

			labels[key] = match
		}
	}

	return labels, nil
}
