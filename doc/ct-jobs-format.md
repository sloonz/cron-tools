# Cron jobs format

Each cron job format is represented by a JSON dictionary containing the
following keys:

* `id`: the job identifier
* `user`
* `host`
* `frequency`: in `crontab(5)` format
* `command`: array
* `tz`: timezone of the server executing the command. Used by `ct digest`, see the documentation associated with it.
* `extra`: a JSON object containing arbitrary key-values, intended to be used by the system administrator or other tools.
