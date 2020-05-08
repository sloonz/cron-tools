# Logs repository format

Logs are stored as simple files and grouped by job identifier: all
logs of the same job identifier are stored in the same directory. Slash
characters (`/`) are permitted in job identifiers and will be mirrored
in the directory hierarchy of the storage.

When the repository is stored in an object store, this hierarchy is
mapped directly to object keys and key prefixes.

Log files are named following this format :

```text
Date.Nonce.ext
```

Where:

* Date is the date contained in the `Date` header. Any colon in the time
part is removed

* Nonce is a unique string ensuring that two job logs having the same
job identifier and start date are in different files

* Extension is optional and indicates compression. Currently only `gz`
is supported by `cron-tools`

Job identifiers should not start with a dot (`.`) character. Files
and directories starting with a dot in the repository are reserved for
implementation-dependent storage and operator.

Job identifiers should not start with an underscore (`_`) character. Files
and directories starting with an underscore are reserved for special
meaning. Currently, there are five of them :

* `_err` stores failure logs, since failure logs do not contain a job
identifier.

* `_invalid` stores ill-formed logs that cannot be parsed.

* `_incoming` is a temporary storage area for incoming logs.

* `_jobs`: files put here contains cron jobs descriptions. While it
is not required, it can be used by the operator to simplify the usage
of `ct digest`.

* `_sync` is a script or executable that should fetch remote logs. It will
be called by the web interface at startup, and when the user request it.
