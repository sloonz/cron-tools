# ct digest

## Synopsis

`**ct digest** [-r */path/to/repository*|s3://bucket/prefix|s3+name://bucket/prefix] [--object-storage-config-file /etc/cron-tools/object-storage.conf] [*jobs-repository* ...]`

## Description

`ct digest` reads cron job descriptions from the `_jobs` directory of one
or more repositories, and for each cron job, looks at its status in the
main cron logs repository. The status of cron jobs are then summarized
in a JSON output that can be processed by a script (for an example, see
`contrib/ct-digest-to-slack`).

If no repository path is provided, it is read from one of those symbolic
links, the first one taking precedence:

* `$HOME/.config/cron-tools/repository`
* `/etc/cron-tools/repository`

The repository can also be an `s3://bucket/prefix` or
`s3+name://bucket/prefix` URL. In that case,
`--object-storage-config-file` defaults to
`/etc/cron-tools/object-storage.conf` and uses the same `rclone`-
compatible file format as described in [ct web](ct-web.md). `s3://`
uses the first usable `s3` section in that file, while `s3+name://`
selects the `name` section explicitly.

Positional arguments are repositories from which `ct digest` should read
job descriptions, using each repository’s `_jobs` directory. If no
positional repository is provided, `ct digest` reads job descriptions
from the `_jobs` directory of the repository passed with `-r`.

## Output format

The output of `ct digest` is an array of summaries. A summary is an
object with those fields :

* `description`: cron job description (id, user, host, command...)
* `previousRunTime`: when the cron job should have been run for the last time
* `nextRunTime`: when the cron job should be run next time
* `lastLog`: status of the last log
  * `logId`: identifier
  * `time`: time encoded in the log identifier
  * `startTime`: execution start timestamp from the `cronlog-v0` marker
  * `endTime`: execution finish timestamp from `Exit` (zero time if absent)
  * `headers`: headers of the log, containing information like the command, user,...
  * `footers`: footers of the log, containing information like exit status

For a more detailed description of `headers` and `footers`, see the [cron logs format](logs-format.md).

`previousRunTime` and `nextRunTime` are based on the `frequency` field
of the cron job description. Since `cron` run jobs based on the local
time, we need to know the configured timezone of the server running the
cron job. This is done by specifying the `tz` field, in the format
described [here](https://golang.org/pkg/time/#LoadLocation). If the
field is mispecified, the predicted `previousRunTime` and `nextRunTime`
will be off.

Do NOT use `Local` for the `tz` field, unless the machine running
`ct digest` has the same configured timezone as your servers.

You can use the difference between `previousRunTime` (predicted last run)
with `lastLog.time` (actual last run) to detect cron jobs that are not
run on schedule, or not running at all anymore.
