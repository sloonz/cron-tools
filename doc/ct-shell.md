# ct shell

## Synopsis

`**ct shell** *command*`

## Rationale

When running a command with `ct run` on a remote host, with the intent
of processing it (for example `ct notify`) or storing it (for example,
with `ct store`), `ct run` will ensure that the output is in the cron
log format :

```shell
# SSH failure will prevent ct notify to work as intended
ssh user@example.com ct run /usr/local/bin/run-backup | ct notify
```

However, if ssh fails for any reason (for example lack of connectivity),
the output will NOT be in the cron logs format and will NOT be usable
by tools like `ct notify`.

`ct shell` is a thin wrapper that will emit an output if the cron log
format if the command it runs does not :

```shell
# SSH failure will be processed as intended by ct notify
ct shell ssh user@example.com ct run /usr/local/bin/run-backup | ct notify
```

## Description

`ct shell` executes the provided command and ensure that its output is a
valid cron log :

* If the command exits without emitting any output, or if the emitted
output does not start with either the required `meta F cronlog-v0` marker line or the magic string `CRONERR\n`,
then a cron log of type `CRONERR` is emitted.

* If the command emits any data on its standard error before emitting
either the required `meta F cronlog-v0` marker line or the magic string `CRONERR\n` on its standard output,
then a cron log of type `CRONERR` is emitted.

* Failure to launch the command will cause a cron log of type `CRONEER`
to be emitted.
