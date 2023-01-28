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
