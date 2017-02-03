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
stern [options] <pod>
```

The `pod` query is a regular expression so you could provide `"web-\w"` to tail
`web-backend` and `web-frontend` pods but not `web-123`.

### cli flags

| flag            | default          | purpose                                                                                  |
|-----------------|------------------|------------------------------------------------------------------------------------------|
| `--container`   | `.*`             | Container name when multiple containers in pod (regular expression)                      |
| `--timestamps`  |                  | Print timestamps                                                                         |
| `--since`       |                  | Return logs newer than a relative duration like 52, 2m, or 3h. Displays all if omitted   |
| `--context`     |                  | Kubernetes context to use. Default to `kubectl config current-context`                   |
| `--exclude`     |                  | Log lines to exclude; specify multiple with additional `--exclude`; (regular expression) |
| `--namespace`   |                  | Kubernetes namespace to use. Default to namespace configured in Kubernetes context       |
| `--kube-config` | `~/.kube/config` | Path to kubeconfig file to use                                                           |

See `stern --help` for details

## Examples:

Tail the `gateway` container running inside of the `envvars` pod on staging
```
stern --context staging --container gateway envvars
```

Show auth activity from 15min ago with timestamps
```
stern -t --since 15m auth
```

Follow the development of `some-new-feature` in minikube
```
stern --context minikube some-new-feature
```

View pods from another namespace
```
stern --namespace kube-system kubernetes-dashboard
```
