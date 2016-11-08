# stern

[![wercker status](https://app.wercker.com/status/fb1ed340ffed75c22dc301c38ab0893c/s/master "wercker status")](https://app.wercker.com/project/byKey/fb1ed340ffed75c22dc301c38ab0893c)

Stern allows you to `tail` multiple pods on Kubernetes and multiple containers within the pod. Each result is color coded for quicker debugging.

The query is a regular expression so the pod name can easily be filtered and you don't need to specify the exact id (for instance omitting the deployment id). If a pod is deleted it gets removed from tail and if a new is added it automatically gets tailed.

When a pod contains multiple containers Stern can tail all of them too without having to do this manually for each one. Simply specify the `container` flag to limit what containers to show. By default all containers are listened to.

## Installation

```sh
go get -u github.com/wercker/stern
```

## Usage

```
NAME:
   stern - Tail multiple pods and containers from Kubernetes

USAGE:
   stern [options] pod-query

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --container value, -c value  Container name when multiple containers in pod (default: ".*")
   --timestamps, -t             Print timestamps
   --since value, -s value      Return logs newer than a realtive duration like 5s, 2m, or 3h. Defaults to all logs.
   --context value              Kubernetes context to use
   --namespace value            Kubernetes namespace to use (default: "default")
   --kube-config value          Path to kubeconfig file to use
   --help, -h                   show help
   --version, -v                print the version
```

### Examples:

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
