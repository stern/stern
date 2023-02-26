[![Build](https://github.com/stern/stern/workflows/CI/badge.svg)](https://github.com/stern/stern/actions?query=workflow%3ACI+branch%3Amaster)
# stern

*Fork of discontinued [wercker/stern](https://github.com/wercker/stern)*

Stern allows you to `tail` multiple pods on Kubernetes and multiple containers
within the pod. Each result is color coded for quicker debugging.

The query is a regular expression or a Kubernetes resource in the form
 `<resource>/<name>` so the pod name can easily be filtered and
you don't need to specify the exact id (for instance omitting the deployment
id). If a pod is deleted it gets removed from tail and if a new pod is added it
automatically gets tailed.

When a pod contains multiple containers Stern can tail all of them too without
having to do this manually for each one. Simply specify the `container` flag to
limit what containers to show. By default all containers are listened to.

## Installation

### Download binary

Download a [binary release](https://github.com/stern/stern/releases)

### Build from source

```
go install github.com/stern/stern@latest
```

### asdf (Linux/macOS)

If you use [asdf](https://asdf-vm.com/), you can install like this:
```
asdf plugin-add stern
asdf install stern latest
```

### Homebrew (Linux/macOS)

If you use [Homebrew](https://brew.sh), you can install like this:
```
brew install stern
```

### Krew (Linux/macOS/Windows)

If you use [Krew](https://krew.sigs.k8s.io/) which is the package manager for kubectl plugins, you can install like this:
```
kubectl krew install stern
```

## Usage

```
stern pod-query [flags]
```

The `pod` query is a regular expression or a Kubernetes resource in the form `<resource>/<name>`.

The query is a regular expression when it is not a Kubernetes resource,
so you could provide `"web-\w"` to tail `web-backend` and `web-frontend` pods but not `web-123`.

When the query is in the form `<resource>/<name>` (exact match), you can select all pods belonging
to the specified Kubernetes resource, such as `deployment/nginx`.
Supported Kubernetes resources are `pod`, `replicationcontroller`, `service`, `daemonset`, `deployment`,
`replicaset`, `statefulset` and `job`.

### cli flags

<!-- auto generated cli flags begin --->
 flag                        | default   | purpose
-----------------------------|-----------|---------
 `--all-namespaces`, `-A`    | `false`   | If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.
 `--color`                   | `auto`    | Force set color output. 'auto':  colorize if tty attached, 'always': always colorize, 'never': never colorize.
 `--completion`              |           | Output stern command-line completion code for the specified shell. Can be 'bash', 'zsh' or 'fish'.
 `--container`, `-c`         | `.*`      | Container name when multiple containers in pod. (regular expression)
 `--container-state`         | `all`     | Tail containers with state in running, waiting, terminated, or all. 'all' matches all container states. To specify multiple states, repeat this or set comma-separated value.
 `--context`                 |           | Kubernetes context to use. Default to current context configured in kubeconfig.
 `--ephemeral-containers`    | `true`    | Include or exclude ephemeral containers.
 `--exclude`, `-e`           | `[]`      | Log lines to exclude. (regular expression)
 `--exclude-container`, `-E` | `[]`      | Container name to exclude when multiple containers in pod. (regular expression)
 `--exclude-pod`             | `[]`      | Pod name to exclude. (regular expression)
 `--field-selector`          |           | Selector (field query) to filter on. If present, default to ".*" for the pod-query.
 `--include`, `-i`           | `[]`      | Log lines to include. (regular expression)
 `--init-containers`         | `true`    | Include or exclude init containers.
 `--kubeconfig`              |           | Path to kubeconfig file to use. Default to KUBECONFIG variable then ~/.kube/config path.
 `--max-log-requests`        | `-1`      | Maximum number of concurrent logs to request. Defaults to 50, but 5 when specifying --no-follow
 `--namespace`, `-n`         |           | Kubernetes namespace to use. Default to namespace configured in kubernetes context. To specify multiple namespaces, repeat this or set comma-separated value.
 `--no-follow`               | `false`   | Exit when all logs have been shown.
 `--only-log-lines`          | `false`   | Print only log lines
 `--output`, `-o`            | `default` | Specify predefined template. Currently support: [default, raw, json, extjson, ppextjson]
 `--prompt`, `-p`            | `false`   | Toggle interactive prompt for selecting 'app.kubernetes.io/instance' label values.
 `--selector`, `-l`          |           | Selector (label query) to filter on. If present, default to ".*" for the pod-query.
 `--since`, `-s`             | `48h0m0s` | Return logs newer than a relative duration like 5s, 2m, or 3h.
 `--tail`                    | `-1`      | The number of lines from the end of the logs to show. Defaults to -1, showing all logs.
 `--template`                |           | Template to use for log lines, leave empty to use --output flag.
 `--template-file`, `-T`     |           | Path to template to use for log lines, leave empty to use --output flag.
 `--timestamps`, `-t`        | `false`   | Print timestamps.
 `--timezone`                | `Local`   | Set timestamps to specific timezone.
 `--verbosity`               | `0`       | Number of the log level verbosity
 `--version`, `-v`           | `false`   | Print the version and exit.
<!-- auto generated cli flags end --->

See `stern --help` for details

Stern will use the `$KUBECONFIG` environment variable if set. If both the
environment variable and `--kubeconfig` flag are passed the cli flag will be
used.

### templates

stern supports outputting custom log messages.  There are a few predefined
templates which you can use by specifying the `--output` flag:

| output    | description                                                                                           |
|-----------|-------------------------------------------------------------------------------------------------------|
| `default` | Displays the namespace, pod and container, and decorates it with color depending on --color           |
| `raw`     | Only outputs the log message itself, useful when your logs are json and you want to pipe them to `jq` |
| `json`    | Marshals the log struct to json. Useful for programmatic purposes                                     |

It accepts a custom template through the `--template` flag, which will be
compiled to a Go template and then used for every log message. This Go template
will receive the following struct:

| property        | type   | description                                 |
|-----------------|--------|---------------------------------------------|
| `Message`       | string | The log message itself                      |
| `NodeName`      | string | The node name where the pod is scheduled on |
| `Namespace`     | string | The namespace of the pod                    |
| `PodName`       | string | The name of the pod                         |
| `ContainerName` | string | The name of the container                   |

The following functions are available within the template (besides the [builtin
functions](https://golang.org/pkg/text/template/#hdr-Functions)):

| func        | arguments             | description                                                     |
|-------------|-----------------------|-----------------------------------------------------------------|
| `json`      | `object`              | Marshal the object and output it as a json text                 |
| `color`     | `color.Color, string` | Wrap the text in color (.ContainerColor and .PodColor provided) |
| `parseJSON` | `string`              | Parse string as JSON                                            |
| `extjson`   | `string`              | Parse the object as json and output colorized json              |
| `ppextjson` | `string`              | Parse the object as json and output pretty-print colorized json |

### Log level verbosity

You can configure the log level verbosity by the `--verbosity` flag.
It is useful when you want to know how stern interacts with a Kubernetes API server in troubleshooting.

Increasing the verbosity increases the number of logs. `--verbosity 6` would be a good starting point.

### Max log requests

Stern has the maximum number of concurrent logs to request to prevent unintentional load to a cluster.
The number can be configured by the `--max-log-requests` flag.

The behavior and the default are different depending on the presence of the `--no-follow` flag.

| `--no-follow` | default | behavior         |
|---------------|---------|------------------|
| specified     | 5       | limits the number of concurrent logs to request |
| not specified | 50      | exits with an error when if it reaches the concurrent limit |

The combination of `--max-log-requests 1` and `--no-follow` will be helpful if you want to show logs in order.

## Examples:

Tail the `gateway` container running inside of the `envvars` pod on staging
```
stern envvars --context staging --container gateway
```

Tail the `staging` namespace excluding logs from `istio-proxy` container
```
stern -n staging --exclude-container istio-proxy .
```

Tail the `kube-system` namespace excluding logs from `kube-apiserver` pod
```
stern -n kube-system --exclude-pod kube-apiserver .
```

Show auth activity from 15min ago with timestamps
```
stern auth -t --since 15m
```

Show auth activity with timestamps in specific timezone (default is your local timezone)
```
stern auth -t --timezone Asia/Tokyo
```

Follow the development of `some-new-feature` in minikube
```
stern some-new-feature --context minikube
```

View pods from another namespace
```
stern kubernetes-dashboard --namespace kube-system
```

Tail the pods filtered by `run=nginx` label selector across all namespaces
```
stern --all-namespaces -l run=nginx
```

Follow the `frontend` pods in canary release
```
stern frontend --selector release=canary
```

Tail the pods on `kind-control-plane` node across all namespaces
```
stern --all-namespaces --field-selector spec.nodeName=kind-control-plane
```

Tail the pods created by `deployment/nginx`
```
stern deployment/nginx
```

Pipe the log message to jq:
```
stern backend -o json | jq .
```

Only output the log message itself:
```
stern backend -o raw
```

Output using a custom template:

```
stern --template '{{printf "%s (%s/%s/%s/%s)\n" .Message .NodeName .Namespace .PodName .ContainerName}}' backend
```

Output using a custom template with stern-provided colors:

```
stern --template '{{.Message}} ({{.Namespace}}/{{color .PodColor .PodName}}/{{color .ContainerColor .ContainerName}}){{"\n"}}' backend
```

Output using a custom template with `parseJSON`:

```
stern --template='{{.PodName}}/{{.ContainerName}} {{with $d := .Message | parseJSON}}[{{$d.level}}] {{$d.message}}{{end}}{{"\n"}}' backend
```

Trigger the interactive prompt to select an 'app.kubernetes.io/instance' label value:

```
stern -p
```

Output log lines only:

```
stern . --only-log-lines
```

## Completion

Stern supports command-line auto completion for bash, zsh or fish. `stern
--completion=(bash|zsh|fish)` outputs the shell completion code which work by being
evaluated in `.bashrc`, etc for the specified shell. In addition, Stern
supports dynamic completion for `--namespace`, `--context`, a resource query
in the form `<resource>/<name>`, and flags with pre-defined choices.

If you use bash, stern bash completion code depends on the
[bash-completion](https://github.com/scop/bash-completion). On the macOS, you
can install it with homebrew as follows:

```
# If running Bash 3.2
brew install bash-completion

# or, if running Bash 4.1+
brew install bash-completion@2
```

Note that bash-completion must be sourced before sourcing the stern bash
completion code in `.bashrc`.

```sh
source "$(brew --prefix)/etc/profile.d/bash_completion.sh"
source <(stern --completion=bash)
```

If you use zsh, just source the stern zsh completion code in `.zshrc`.

```sh
source <(stern --completion=zsh)
```

if you use fish shell, just source the stern fish completion code.

```sh
stern --completion=fish | source

# To load completions for each session, execute once:
stern --completion=fish >~/.config/fish/completions/stern.fish
```

## Running with container

You can also use stern using a container:

```
docker run ghcr.io/stern/stern --version
```

If you are using a minikube cluster, you need to run a container as follows:

```
docker run --rm -v "$HOME/.minikube:$HOME/.minikube" -v "$HOME/.kube:/$HOME/.kube" -e KUBECONFIG="$HOME/.kube/config" ghcr.io/stern/stern .
```

You can find image tags in https://github.com/orgs/stern/packages/container/package/stern.

## Running in Kubernetes Pods

If you want to use stern in Kubernetes Pods, you need to create the following ClusterRole and bind it to ServiceAccount.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: stern
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "watch", "list"]
```

## Contributing to this repository

Please see [CONTRIBUTING](CONTRIBUTING.md) for details.
