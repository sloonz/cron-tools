# ct web

## Synopsis

`ct web [-r /path/to/repository|s3://bucket/prefix|s3+name://bucket/prefix] [-l 127.0.0.1:3000] [--cache /path/to/cache.db] [--object-storage-config-file /etc/cron-tools/object-storage.conf] [*cron-job-descriptions*]`

## Description

`ct web` offers a web interface to visualize the logs contained in a
cron logs repository.

It does not provide any authentication method ; this should be handled
by a reverse proxy like nginx.

If no repository path is provided, it is read from one of those symbolic
links, the first one taking precedence:

* `$HOME/.config/cron-tools/repository`
* `/etc/cron-tools/repository`

The repository can also be an `s3://bucket/prefix` or
`s3+name://bucket/prefix` URL. In that case, `ct web` reads logs and
`_jobs` files from the object store using the same logical repository
layout. `--object-storage-config-file` defaults to
`/etc/cron-tools/object-storage.conf` and provides the connection
settings and credentials. `ct web` always uses path-style bucket
addressing for object storage repositories.

The object storage configuration file uses an `rclone`-compatible INI
layout. If a section has no `type`, it defaults to `s3`. If a section
has a `type` and it is not `s3`, that section is ignored by cron-tools.
The `provider` and `env_auth` keys are ignored. Supported keys are:

* `type`
* `access_key_id`
* `secret_access_key`
* `region`
* `endpoint`

Example:

```text
[storage]
type = s3
provider = Other
env_auth = false
access_key_id = example-access-key
secret_access_key = example-secret-key
region = us-east-1
endpoint = https://objects.example.com
```

`s3+storage://bucket/prefix` selects the `storage` section explicitly.
Plain `s3://bucket/prefix` uses the first usable `s3` section in the
configuration file.

You can also pass as an argument a list of cron job descriptions (in
the same way as described in [ct gen](ct-gen.md)). If none is given, the
`_jobs` directory inside the repository will be used. This list will be
used to display cron jobs that are never been runned (and therefore have
no logs) or signal jobs that have not been run on schedule.

If a `_sync` script or executable is present in the repository, the
script is called every time an user starts the interface, and a button
is provided to the user in the interface to run it. Its job should be
to fetch logs on remote servers. This only applies to local filesystem
repositories.

If `--cache` is not provided, the metadata cache defaults to
`REPOSITORY/.cache` for local repositories and to a file under the user
cache directory for object storage repositories.
