# cron-tools

This project aims to bring advanced features (central logging, dashboard,
authorization and alerts) to `cron(8)`. It is a set of formats and tools
aiming to be used with cron, not to replace it, in order to improve
system administrators experience.

`cron-tools` is :

* [A format for cron run logs](doc/logs-format.md)
* [A format for a repository of logs](doc/repo-format.md)
* [A format to describe cron jobs](doc/jobs-format.md)
* [`ct run`](doc/ct-run.md), a tool to generate logs in the aformentioned format
* [`ct store`](doc/ct-store.md), a tool to write a log in the store
* [`ct web`](doc/ct-web.md), a simple web dashboard
* [`ct digest`](doc/ct-digest.md), a tool to create a periodic summary of cron status
* [`ct gen`](doc/ct-gen.md), a tool to create crontabs
* [`ct notify`](doc/ct-notify.md), a tool to send alerts for failed crons
* [`ct shell`](doc/ct-shell.md), a wrapper to handle unexpected errors

`cron-tools` can be used as building blocks to create a variety of
workflows. For example, you can have a centralized crontab on one server
that will use SSH to launch jobs in a secure way on other servers (jobs
can be signed offline so an attacker gaining access to the central
server won’t gain the ability to launch arbitrary commands on other
servers). Or you can have a centralized logs repository, with each
server sending its logs to it. Or you can have one crontab and one logs
repository per server, and aggregate them with `rsync`. Or you can decide
that you don't need a cron logs repository and just do with `ct run`
and `ct notify` to get a notification on failed jobs.

## Development

This repository uses Go modules. To build and test the Go code:

```sh
make all
go test ./ ./cmd/... ./pkg ./web
```

To regenerate module metadata after dependency changes:

```sh
go mod tidy
```
