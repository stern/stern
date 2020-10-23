# This repository is a friendly fork of https://github.com/wercker/stern - discontinued due to lack of maintainers.

# stern

Stern allows you to `tail` multiple pods on Kubernetes and multiple containers
within the pod. Each result is color coded for quicker debugging.

The query is a regular expression so the pod name can easily be filtered and
you don't need to specify the exact id (for instance omitting the deployment
id). If a pod is deleted it gets removed from tail and if a new pod is added it
automatically gets tailed.

When a pod contains multiple containers Stern can tail all of them too without
having to do this manually for each one. Simply specify the `container` flag to
limit what containers to show. By default all containers are listened to.

## Installation

If you don't want to build from source go grab a [binary release](https://github.com/stern/stern/releases)

[Govendor](https://github.com/kardianos/govendor) is required to install vendored dependencies.

```
git clone https://github.com/stern/stern.git && cd stern
make install
```

## Usage

```
stern pod-query [flags]
```

The `pod` query is a regular expression so you could provide `"web-\w"` to tail
`web-backend` and `web-frontend` pods but not `web-123`.

### cli flags

| flag                 | default          | purpose                                                                                                      |
|----------------------|------------------|--------------------------------------------------------------------------------------------------------------|
| `--container`        | `.*`             | Container name when multiple containers in pod (regular expression)                                          |
| `--exclude-container`|                  | Container name to exclude when multiple containers in pod (regular expression)                               |
| `--container-state`  | `running`        | Tail containers with status in running, waiting or terminated. Default to running.                           |
| `--timestamps`       |                  | Print timestamps                                                                                             |
| `--since`            |                  | Return logs newer than a relative duration like 52, 2m, or 3h. Displays all if omitted                       |
| `--context`          |                  | Kubernetes context to use. Default to `kubectl config current-context`                                       |
| `--exclude`          |                  | Log lines to exclude; specify multiple with additional `--exclude`; (regular expression)                     |
| `--namespace`        |                  | Kubernetes namespace to use. Default to namespace configured in Kubernetes context                           |
| `--kubeconfig`       | `~/.kube/config` | Path to kubeconfig file to use                                                                               |
| `--all-namespaces`   |                  | If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.  |
| `--selector`         |                  | Selector (label query) to filter on. If present, default to `.*` for the pod-query.                          |
| `--tail`             | `-1`             | The number of lines from the end of the logs to show. Defaults to -1, showing all logs.                      |
| `--color`            | `auto`           | Force set color output. `auto`: colorize if tty attached, `always`: always colorize, `never`: never colorize |
| `--output`           | `default`        | Specify predefined template. Currently support: [default, raw, json] See templates section                   |
| `template`           |                  | Template to use for log lines, leave empty to use --output flag                                              |

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

| property        | type   | description               |
|-----------------|--------|---------------------------|
| `Message`       | string | The log message itself    |
| `Namespace`     | string | The namespace of the pod  |
| `PodName`       | string | The name of the pod       |
| `ContainerName` | string | The name of the container |

The following functions are available within the template (besides the [builtin
functions](https://golang.org/pkg/text/template/#hdr-Functions)):

| func    | arguments             | description                                                     |
|---------|-----------------------|-----------------------------------------------------------------|
| `json`  | `object`              | Marshal the object and output it as a json text                 |
| `color` | `color.Color, string` | Wrap the text in color (.ContainerColor and .PodColor provided) |



## Examples:

Tail the `gateway` container running inside of the `envvars` pod on staging
```
stern envvars --context staging --container gateway
```

Tail the `staging` namespace excluding logs from `istio-proxy` container
```
stern -n staging --exclude-container istio-proxy .
```

Show auth activity from 15min ago with timestamps
```
stern auth -t --since 15m
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
stern --template '{{.Message}} ({{.Namespace}}/{{.PodName}}/{{.ContainerName}})' backend
```

Output using a custom template with stern-provided colors:

```
stern --template '{{.Message}} ({{.Namespace}}/{{color .PodColor .PodName}}/{{color .ContainerColor .ContainerName}})' backend
```

## Completion

Stern supports command-line auto completion for bash or zsh. `stern
--completion=(bash|zsh)` outputs the shell completion code which work by being
evaluated in `.bashrc`, etc for the specified shell. In addition, Stern
supports dynamic completion for `--namespace` and `--context`. In order to use
that, kubectl must be installed on your environment.

If you use bash, stern bash completion code depends on the
[bash-completion](https://github.com/scop/bash-completion). On the macOS, you
can install it with homebrew as follows:

```
$ brew install bash-completion
```

Note that bash-completion must be sourced before sourcing the stern bash
completion code in `.bashrc`.

```sh
source <(brew --prefix)/etc/bash-completion
source <(stern --completion=bash)
```

If you use zsh, just source the stern zsh completion code in `.zshrc`.

```sh
source <(stern --completion=zsh)
```

## Contributing to this repository

Please see [CONTRIBUTING](CONTRIBUTING.md) for details.
