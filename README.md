# stern

[![wercker status](https://app.wercker.com/status/fb1ed340ffed75c22dc301c38ab0893c/s/master "wercker status")](https://app.wercker.com/project/byKey/fb1ed340ffed75c22dc301c38ab0893c)

Stern allows you to `tail` multiple pods on Kubernetes and multiple containers
within the pod. Each result is color coded for quicker debugging.

The query is a regular expression so the pod name can easily be filtered and
you don't need to specify the exact id (for instance omitting the deployment
id). If a pod is deleted it gets removed from tail and if a new is added it
automatically gets tailed.

When a pod contains multiple containers Stern can tail all of them too without
having to do this manually for each one. Simply specify the `container` flag to
limit what containers to show. By default all containers are listened to.

## Installation

If you don't want to build from source go grab a [binary release](https://github.com/wercker/stern/releases)

[Govendor](https://github.com/kardianos/govendor) is required to install vendored dependencies.

```
mkdir -p $GOPATH/src/github.com/wercker
cd $GOPATH/src/github.com/wercker
git clone git@github.com:wercker/stern.git && cd stern
govendor sync
go install
```

## Usage

```
stern pod-query [flags]
```

The `pod` query is a regular expression so you could provide `"web-\w"` to tail
`web-backend` and `web-frontend` pods but not `web-123`.

### cli flags

| flag               | default          | purpose                                                                                                      |
|--------------------|------------------|--------------------------------------------------------------------------------------------------------------|
| `--container`      | `.*`             | Container name when multiple containers in pod (regular expression)                                          |
| `--timestamps`     |                  | Print timestamps                                                                                             |
| `--since`          |                  | Return logs newer than a relative duration like 52, 2m, or 3h. Displays all if omitted                       |
| `--context`        |                  | Kubernetes context to use. Default to `kubectl config current-context`                                       |
| `--exclude`        |                  | Log lines to exclude; specify multiple with additional `--exclude`; (regular expression)                     |
| `--namespace`      |                  | Kubernetes namespace to use. Default to namespace configured in Kubernetes context                           |
| `--kube-config`    | `~/.kube/config` | Path to kubeconfig file to use                                                                               |
| `--all-namespaces` |                  | If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.  |
| `--selector`       |                  | Selector (label query) to filter on. If present, default to `.*` for the pod-query.                          |
| `--tail`           | `-1`             | The number of lines from the end of the logs to show. Defaults to -1, showing all logs.                      |
| `--color`          | `auto`           | Force set color output. `auto`: colorize if tty attached, `always`: always colorize, `never`: never colorize |

See `stern --help` for details

## Examples:

Tail the `gateway` container running inside of the `envvars` pod on staging
```
stern envvars --context staging --container gateway
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
