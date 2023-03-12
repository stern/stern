# v1.24.0

## :zap: Nortable Changes

### Add a short format for timestamps

`--timestamps` flag now accepts a format, one of `default` or `short`.

- `default`: the original format `2006-01-02T15:04:05.000000000Z07:00` (RFC3339Nano with trailing zeros)
- `short`: the new format `01-02 15:04:05` (time.DateTime without year).

If `--timestamps` is specified but without value, `default` is used to maintain backward compatibility.

```
$ stern --timestamps=short -n kube-system ds/kindnet --no-follow --tail 1 --only-log-lines
kindnet-hqn2k kindnet-cni 03-12 09:29:53 I0312 00:29:53.620499       1 main.go:250] Node kind-worker3 has CIDR [10.244.1.0/24]
kindnet-5f4ms kindnet-cni 03-12 09:29:53 I0312 00:29:53.374482       1 main.go:250] Node kind-worker3 has CIDR [10.244.1.0/24]
```

### Add `--node` flag to filter on a specific node

New `--node` flag allows you to filter pods on a specific node. This flag will be helpful when we debug pods on the specific node.

```
# Print a DaemonSet pod on the specific node
stern --node <NODE_NAME> daemonsets/<DS_NAME>

# Print all pods on the specific node
stern --node <NODE_NAME> --all-namespaces --no-follow --max-log-requests 1 .
```

### Highlight matched strings in the log lines with the include option

`--include` flag now highlight matched strings in the log lines.

```
stern --include "\[error\]" .
```

### Add `all` option to `--container-state` flag

`--container-state` flag now accepts `all` that is the same with specifying `running,waiting,terminated`. This change is helpful when we debug CrashLoopBackoff containers.

```
# Before
stern --container-state running,terminated,running <QUERY>

# After
stern --container-state all <QUERY>`
```

## :warning: Breaking Changes

### Add `--max-log-requests` flag to limit concurrent requests

New `--max-log-requests` flag allows you to limit concurrent requests to prevent unintentional load to a cluster. The behavior and the default are different depending on the presence of the `--no-follow` flag.

| `--no-follow` | default | behavior         |
|---------------|---------|------------------|
| specified     | 5       | limits the number of concurrent logs to request |
| not specified | 50      | exits with an error when if it reaches the concurrent limit |

If you want to change to the same behavior as before, specify a sufficiently large value for `--max-log-requests`.

### Change the default of `--container-state` flag to `all`

The default value of `--container-state` has been changed to `all` from `running`. With this change, stern will now show logs of completed (`terminated`) and CrashLoopBackoff (`waiting`) pods in addition to running pods by default.

If you want to change to the same behavior as before, explicitly specify `--container-state` to `running`.

## Changes

* Upgrade golang.org/x/net to fix a dependabot alert ([#250](https://github.com/stern/stern/pull/250)) [e26d049](https://github.com/stern/stern/commit/e26d049) (Kazuki Suda)
* Add a short format for timestamps ([#249](https://github.com/stern/stern/pull/249)) [43ab3f1](https://github.com/stern/stern/commit/43ab3f1) (Takashi Kusumi)
* Bump golangci-lint to v1.51.2 ([#248](https://github.com/stern/stern/pull/248)) [079d158](https://github.com/stern/stern/commit/079d158) (Takashi Kusumi)
* Add dynamic completion for --node flag ([#244](https://github.com/stern/stern/pull/244)) [59d4453](https://github.com/stern/stern/commit/59d4453) (Takashi Kusumi)
* Add --node flag to filter on a specific node ([#243](https://github.com/stern/stern/pull/243)) [f90f70f](https://github.com/stern/stern/commit/f90f70f) (Takashi Kusumi)
* allow flexible log parsing and formatting ([#239](https://github.com/stern/stern/pull/239)) [12a55fa](https://github.com/stern/stern/commit/12a55fa) (Dmytro Milinevskyi)
* Documenting how to get Bash completion in Krew mode ([#240](https://github.com/stern/stern/pull/240)) [24c8716](https://github.com/stern/stern/commit/24c8716) (Jesse Glick)
* Add CI for skipped files ([#241](https://github.com/stern/stern/pull/241)) [7131af2](https://github.com/stern/stern/commit/7131af2) (Takashi Kusumi)
* Replace actions/cache with setup-go's cache ([#238](https://github.com/stern/stern/pull/238)) [74952fd](https://github.com/stern/stern/commit/74952fd) (Takashi Kusumi)
* Make CI jobs faster ([#237](https://github.com/stern/stern/pull/237)) [4bb340d](https://github.com/stern/stern/commit/4bb340d) (Kazuki Suda)
* Refactor options.sternConfig() ([#236](https://github.com/stern/stern/pull/236)) [2315b23](https://github.com/stern/stern/commit/2315b23) (Takashi Kusumi)
* Return error when output option is invalid ([#235](https://github.com/stern/stern/pull/235)) [1c5aa2b](https://github.com/stern/stern/commit/1c5aa2b) (Takashi Kusumi)
* Refactor template logic ([#233](https://github.com/stern/stern/pull/233)) [371daf1](https://github.com/stern/stern/commit/371daf1) (Takashi Kusumi)
* Revert "add support to parse JSON logs ([#228](https://github.com/stern/stern/pull/228))" ([#232](https://github.com/stern/stern/pull/232)) [202f7e8](https://github.com/stern/stern/commit/202f7e8) (Dmytro Milinevskyi)
* Change the default of --container-state to `all` ([#225](https://github.com/stern/stern/pull/225)) [2502c91](https://github.com/stern/stern/commit/2502c91) (Takashi Kusumi)
* Highlight matched strings in the log lines with the include option ([#231](https://github.com/stern/stern/pull/231)) [9fbaa18](https://github.com/stern/stern/commit/9fbaa18) (Kazuki Suda)
* Support resuming from the last log when retrying ([#230](https://github.com/stern/stern/pull/230)) [52894f8](https://github.com/stern/stern/commit/52894f8) (Takashi Kusumi)
* add support to parse JSON logs ([#228](https://github.com/stern/stern/pull/228)) [72a5854](https://github.com/stern/stern/commit/72a5854) (Dmytro Milinevskyi)
* Show initContainers first when --no-follow and --max-log-requests 1 ([#226](https://github.com/stern/stern/pull/226)) [ef753f1](https://github.com/stern/stern/commit/ef753f1) (Takashi Kusumi)
* Add --max-log-requests flag to limit concurrent requests ([#224](https://github.com/stern/stern/pull/224)) [0b939c5](https://github.com/stern/stern/commit/0b939c5) (Takashi Kusumi)
* Improve handling of container termination ([#221](https://github.com/stern/stern/pull/221)) [8312782](https://github.com/stern/stern/commit/8312782) (Takashi Kusumi)
* Allow pods without labels to be selected in the resource query ([#223](https://github.com/stern/stern/pull/223)) [fc51906](https://github.com/stern/stern/commit/fc51906) (Takashi Kusumi)
* Add `all` option to --container-state flag ([#222](https://github.com/stern/stern/pull/222)) [6e0d5fc](https://github.com/stern/stern/commit/6e0d5fc) (Takashi Kusumi)

# v1.23.0

## New features

### Add `--no-follow` flag to exit when all logs have been shown

New `--no-follow` flag allows you to exit when all logs have been shown.

```
stern --no-follow .
```

### Support `<resource>/<name>` form as a query

Stern now supports a Kubernetes resource query in the form `<resource>/<name>`. Pod query can still be used.

```
stern deployment/nginx
```

The following Kubernetes resources are supported:

- daemonset
- deployment
- job
- pod
- replicaset
- replicationcontroller
- service
- statefulset

Shell completion of stern already supports this feature.

### Add --verbosity flag to set log level verbosity

New `--verbosity` flag allows you to set the log level verbosity of Kubernetes client-go. This feature is useful when you want to know how stern interacts with a Kubernetes API server in troubleshooting.

```
stern --verbosity=6 .
```

### Add --only-log-lines flag to print only log lines

New `--only-log-lines` flag allows you to print only log lines (and errors if occur). The difference between not specifying the flag and specifying it is as follows:

```
$ stern . --tail=1 --no-follow
+ nginx-cfbcb7b98-96xsv › nginx
+ nginx-cfbcb7b98-29wn7 › nginx
nginx-cfbcb7b98-96xsv nginx 2023/01/27 13:20:48 [notice] 1#1: start worker process 46
- nginx-cfbcb7b98-96xsv › nginx
nginx-cfbcb7b98-29wn7 nginx 2023/01/27 13:20:45 [notice] 1#1: start worker process 46
- nginx-cfbcb7b98-29wn7 › nginx

$ stern . --tail=1 --no-follow --only-log-lines
nginx-cfbcb7b98-96xsv nginx 2023/01/27 13:20:48 [notice] 1#1: start worker process 46
nginx-cfbcb7b98-29wn7 nginx 2023/01/27 13:20:45 [notice] 1#1: start worker process 46
```

## Changes

* Allow to specify --exclude-pod/container multiple times ([#218](https://github.com/stern/stern/pull/218)) [b04478c](https://github.com/stern/stern/commit/b04478c) (Kazuki Suda)
* Add --only-log-lines flag that prints only log lines ([#216](https://github.com/stern/stern/pull/216)) [995be39](https://github.com/stern/stern/commit/995be39) (Kazuki Suda)
* Fix typo of --verbosity flag ([#215](https://github.com/stern/stern/pull/215)) [6c6db1d](https://github.com/stern/stern/commit/6c6db1d) (Takashi Kusumi)
* Add --verbosity flag to set log level verbosity ([#214](https://github.com/stern/stern/pull/214)) [5327626](https://github.com/stern/stern/commit/5327626) (Takashi Kusumi)
* Add completion for flags with pre-defined choices ([#211](https://github.com/stern/stern/pull/211)) [e03646c](https://github.com/stern/stern/commit/e03646c) (Takashi Kusumi)
* Fix bug where container-state is ignored when no-follow specified ([#210](https://github.com/stern/stern/pull/210)) [1bbee8c](https://github.com/stern/stern/commit/1bbee8c) (Takashi Kusumi)
* Add dynamic completion for a resource query ([#209](https://github.com/stern/stern/pull/209)) [2983c8f](https://github.com/stern/stern/commit/2983c8f) (Takashi Kusumi)
* Support `<resource>/<name>` form as a query ([#208](https://github.com/stern/stern/pull/208)) [7bc45f0](https://github.com/stern/stern/commit/7bc45f0) (Takashi Kusumi)
* Fix indent in update-readme.go ([#207](https://github.com/stern/stern/pull/207)) [daf2464](https://github.com/stern/stern/commit/daf2464) (Takashi Kusumi)
* Update dependencies and tools ([#205](https://github.com/stern/stern/pull/205)) [1bcb576](https://github.com/stern/stern/commit/1bcb576) (Kazuki Suda)
* Add --no-follow flag to exit when all logs have been shown ([#204](https://github.com/stern/stern/pull/204)) [a5e581d](https://github.com/stern/stern/commit/a5e581d) (Takashi Kusumi)
* Use StringArrayVarP for --include and --exclude flags ([#196](https://github.com/stern/stern/pull/196)) [80a68a9](https://github.com/stern/stern/commit/80a68a9) (partcyborg)
* Fix the invalid command in README.md ([#193](https://github.com/stern/stern/pull/193)) [f6e76ba](https://github.com/stern/stern/commit/f6e76ba) (Kazuki Suda)
