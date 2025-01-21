# v1.32.0

## :zap: Notable Changes

### A new template function `prettyJSON`

You can now use a new template function `prettyJSON` that parse input and emit it as pretty printed JSON. If it parse fails output string as is.

```
# Will try to parse .Message as JSON and pretty print it, if not json will output as is
stern --template='{{ .Message | prettyJSON }}{{"\n"}}' backend
# Or with parsed json, will drop non-json logs because of `with`
stern --template='{{ with $msg := .Message | tryParseJSON }}{{ prettyJSON $msg }}{{"\n"}}{{end}}' backend
```

### A new template function `bunyanLevelColor`

You can now use a new template function `bunyanLevelColor` that print [bunyan](https://github.com/trentm/node-bunyan) numeric log level using appropriate color.

### A new flag `--condition`

A new `--condition` allows you to filter logs with the pod condition on: `[condition-name[=condition-value]`. The default condition-value is true. Match is case-insensitive. Currently, it is only supported with --tail=0 or --no-follow.

```
# Only display logs for pods that are not ready:
stern . --condition=ready=false --tail=0
```

## Changes

* Add `--condition` (#276) 2576972 (Felipe Santos)
* Add check for when `--no-follow` is set with `--tail=0` (#331) 276e906 (Felipe Santos)
* Implement JSON pretty print (#324) ccd8add (Fabio Napoleoni)
* Fix descriptions of `extjson` and `ppextjson` (#325) d9a9858 (Takashi Kusumi)
* Allow `levelColor` template function to parse numbers (#321) db69276 (Jimmie Högklint)

# v1.31.0

## Changes
* Fix --verbosity flag to show missing logs ([#317](https://github.com/stern/stern/pull/317)) c2b4410 (Takashi Kusumi)
* Update dependencies for Kubernetes 1.31 ([#315](https://github.com/stern/stern/pull/315)) a4fdcc9 (Takashi Kusumi)

# v1.30.0

## :zap: Notable Changes

### Add support for configuring colors for pods and containers
You can now configure highlight colors for pods and containers in [the config file](https://github.com/stern/stern/blob/master/README.md#config-file) using a comma-separated list of [SGR (Select Graphic Rendition) sequences](https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters). See the ["Customize highlight colors" section](https://github.com/stern/stern/blob/master/README.md#customize-highlight-colors) for details.

Example configuration:

```yaml
# Green, Yellow, Blue, Magenta, Cyan, White
pod-colors: "32,33,34,35,36,37"

# Colors with underline (4)
# If empty, the pod colors will be used as container colors
container-colors: "32;4,33;4,34;4,35;4,36;4,37;4"
```

### Display different colors for different containers
A new `--diff-container` flag allows displaying different colors for different containers. This is useful when you want to debug logs for multiple containers in the same pod.

You can also enable this feature in [the config file](https://github.com/stern/stern/blob/master/README.md#config-file).

```yaml
diff-container: true
```

## Changes
* Add support to configure colors for pods and containers ([#306](https://github.com/stern/stern/pull/306)) [f4b2edc](https://github.com/stern/stern/commit/f4b2edc) (Takashi Kusumi)
* Display different colors for different containers ([#305](https://github.com/stern/stern/pull/305)) [d1b5d74](https://github.com/stern/stern/commit/d1b5d74) (Se7en)
* Support an array value in the config file ([#303](https://github.com/stern/stern/pull/303)) [6afabde](https://github.com/stern/stern/commit/6afabde) (Takashi Kusumi)

# v1.29.0

## :zap: Notable Changes

### A new `--stdin` flag for parsing logs from stdin

A new `--stdin` flag has been added, allowing parsing logs from stdin. This flag is helpful when applying the same template to local logs.

```
stern --stdin --template \
  '{{with $msg := .Message | tryParseJSON}}{{toTimestamp $msg.ts "01-02 15:04:05" "Asia/Tokyo"}} {{$msg.msg}}{{"\n"}}{{end}}' \
  <etcd.log
```

Additionally, this feature helps test your template with arbitrary logs.

```
stern --stdin --template \
  '{{with $msg := .Message | tryParseJSON}}{{levelColor $msg.level}} {{$msg.msg}}{{"\n"}}{{end}}' <<EOF
{"level":"info","msg":"info message"}
{"level":"error","msg":"error message"}
EOF
```

### Add support for UNIX time with nanoseconds to template functions

The following template functions now support UNIX time seconds with nanoseconds (e.g., `1136171056.02`).

- `toRFC3339Nano`
- `toUTC`
- `toTimestamp`

## Changes

* Add support for UNIX time with nanoseconds to template functions ([#300](https://github.com/stern/stern/pull/300)) 0d580ff (Takashi Kusumi)
* Clarify that '=' cannot be omitted in --timestamps ([#296](https://github.com/stern/stern/pull/296)) ac36420 (Takashi Kusumi)
* Added example to README ([#295](https://github.com/stern/stern/pull/295)) c1649ca (Thomas Güttler)
* Update dependencies for Kubernetes 1.30 ([#293](https://github.com/stern/stern/pull/293)) d82cc9f (Kazuki Suda)
* Add `--stdin` for `stdin` log parsing ([#292](https://github.com/stern/stern/pull/292)) 53fc746 (Jimmie Högklint)

# v1.28.0

## :zap: Notable Changes

### Highlight matched strings in the log lines with the highlight option

Some part of a log line can be highlighted while still displaying all other logs lines.

`--highlight` flag now highlight matched strings in the log lines.

```
stern --highlight "\[error\]" .
```


# v1.27.0

## :zap: Notable Changes

### Add new template function: `toTimestamp`

The `toTimestamp` function takes in an object, a layout, and optionally a timezone. This allows for more custom time parsing, for instance, if a user doesn't care about seeing the date of the log and only the time (in their own timezone) they can use a template such as:

```
--template '{{ with $msg := .Message | tryParseJSON }}[{{ toTimestamp $msg.time "15:04:05" "Local" }}] {{ $msg.msg }}{{ end }}{{ "\n" }}'
```

### Add generic kubectl options

stern now has the generic options that kubectl has, and a new `--show-hidden-options` option.

```
$ stern --show-hidden-options
The following options can also be used in stern:
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "/home/ksuda/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --disable-compression            If true, opt-out of response compression for all requests to the server
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

The number of kubectl generic options is so large that it makes it difficult to see stern's own list of options, so we usually hide them. Use `--show-hidden-options` if you want to list.

## Changes

* Add generic cli options (#283) f315819 (Kazuki Suda)
* 281: Support toTimestamp template function (#282) 5445cd5 (Will Anderson)

# v1.26.0

## :zap: Notable Changes

### Add new template functions

The following template functions have been added in v1.26.0:

- `extractJSONParts`: Parse string as JSON and concatenate the given keys
- `tryExtractJSONParts`: Attempt to parse string as JSON and concatenate the given keys, returning text on failure

## Changes

* Fix the release workflow ([#275](https://github.com/stern/stern/pull/275)) [91d4cd6](https://github.com/stern/stern/commit/91d4cd6) (Kazuki Suda)
* Update dependencies and tools ([#273](https://github.com/stern/stern/pull/273)) [cb94677](https://github.com/stern/stern/commit/cb94677) (Takashi Kusumi)
* Possibility to extract parts of a json-message. ([#271](https://github.com/stern/stern/pull/271)) [d49142c](https://github.com/stern/stern/commit/d49142c) (Niels)
* Fix potential panic in stern.Run() ([#267](https://github.com/stern/stern/pull/267)) [dcba2dd](https://github.com/stern/stern/commit/dcba2dd) (Takashi Kusumi)
* Add log level color keys and handle default ([#264](https://github.com/stern/stern/pull/264)) [65204cc](https://github.com/stern/stern/commit/65204cc) (Jimmie Högklint)
* Fix typo in README.md ([#261](https://github.com/stern/stern/pull/261)) [d7d5a4f](https://github.com/stern/stern/commit/d7d5a4f) (Will May)
* Integrate fmt and vet checks into golangci-lint ([#260](https://github.com/stern/stern/pull/260)) [1d242bc](https://github.com/stern/stern/commit/1d242bc) (Takashi Kusumi)
* Update Github Actions dependencies ([#259](https://github.com/stern/stern/pull/259)) [9e833da](https://github.com/stern/stern/commit/9e833da) (Takashi Kusumi)

# v1.25.0

## :zap: Notable Changes

### Add support for the config file

You can now use the config file to change the default values of stern options. The default config file path is `~/.config/stern/config.yaml`.

```yaml
# <flag name>: <value>
tail: 10
max-log-requests: 999
timestamps: short
```

You can change the config file path with `--config` flag or `STERNCONFIG` environment variable.

## Changes

* Fix the heading level in README.md ([#257](https://github.com/stern/stern/pull/257)) [c2290b4](https://github.com/stern/stern/commit/c2290b4) (Kazuki Suda)
* Update dependencies and tools ([#256](https://github.com/stern/stern/pull/256)) [531f869](https://github.com/stern/stern/commit/531f869) (Kazuki Suda)
* Allow an empty config file ([#255](https://github.com/stern/stern/pull/255)) [c76ea87](https://github.com/stern/stern/commit/c76ea87) (Takashi Kusumi)
* Add support for the config file ([#254](https://github.com/stern/stern/pull/254)) [2fdc298](https://github.com/stern/stern/commit/2fdc298) (Kazuki Suda)
* Make setup-go get Go version from go.mod ([#253](https://github.com/stern/stern/pull/253)) [23feff7](https://github.com/stern/stern/commit/23feff7) (Takashi Kusumi)

# v1.24.0

## :zap: Notable Changes

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
