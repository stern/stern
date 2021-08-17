module github.com/stern/stern

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.2.16
	github.com/fatih/color v1.12.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/cli-runtime v0.22.0
	k8s.io/client-go v0.22.0
	k8s.io/klog/v2 v2.10.0 // indirect
)

// Workaround to deal with https://github.com/kubernetes/klog/issues/253
// Should be deleted when https://github.com/kubernetes/klog/pull/242 is merged and released
replace github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
