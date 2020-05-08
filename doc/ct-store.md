# ct store

## Synopsis

```
**ct run** *command* | **ct store** [--object-storage-config-file /etc/cron-tools/object-storage.conf] [*repository* ...]
```

## Description

`ct store` receives a cron log on its standard input, as formatted by
`ct run`, and stores it in one or more cron repositories. The cron log is still emitted
on its standard output, so you can chain `ct store` and other tools like
`ct notify`.

If no repository is provided, it is read from one of those symbolic
links, the first one taking precedence:

* `$HOME/.config/cron-tools/repository`
* `/etc/cron-tools/repository`

Repositories can also be `s3://bucket/prefix` or
`s3+name://bucket/prefix` URLs. In that case,
`--object-storage-config-file` defaults to
`/etc/cron-tools/object-storage.conf` and uses the same `rclone`-
compatible file format as described in [ct web](ct-web.md). `s3://`
uses the first usable `s3` section in that file, while `s3+name://`
selects the `name` section explicitly.

If several repositories are provided, `ct store` writes the same log to
all of them. A failure on one repository does not stop writes to the
others, but `ct store` will still exit with a failure status after all
destinations have been attempted.

## Usage with SSH

You can locally run a command and store the log in a remote server :

```
ct run -I myserver/root/backup /usr/local/bin/backup | ssh cronsrepo@cronserver ct store
```

However, that means that the local user can connect on your cron
repository server and execute any command. A way to avoid this is to
allow the key of your local user to only run `ct store` on the remote
host. In the `~/.ssh/authorized_keys` file of your remote server, add
the key of your local user in the following way:

```
command="/usr/bin/ct store /home/cronsrepo/repository" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJnuyYa0lNqAXGVLNdWAtWLNShmmOsFK9XpMJuJUjh9e
```

The previous command is then simplified as :

```
# Notice the absence of command passed to SSH
ct run -I myserver/root/backup /usr/local/bin/backup | ssh cronsrepo@cronserver
```
