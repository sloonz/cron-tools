# Templates format for cron-tools

In all places where `cron-tools` is using templates as user-defined
input (for example in `ct notify`), it is using the Go template format
as described here :

https://golang.org/pkg/text/template/

It also includes the [Sprig
Functions](http://masterminds.github.io/sprig/), as well as some
additional functions :

* `shescape` will make a string usable as a single shell argument:
`echo {{shescape "hello world"}}`

* `shjoin` will make a list of strings usable as a list of shell arguments

* `eval` is similar to standard `template` directive, but is a function
instead of a directive, which allows you to use it in a pipeline : `{{eval
"subtemplate" . | shescape}}` will work where `{{template "subtemplate"
. | shescape}}` is not syntactically allowed

* `hostname` returns the local system hostname as seen by the current
process

* `readfile` reads a local file and returns its contents as a string:
`{{get (readfile "/etc/cron-tools/secrets.json" | fromJson) "slack_secret"}}`

* `timeFormats` exposes Go's named time layouts, such as `RFC3339`,
`RFC3339Nano`, `DateTime`, `DateOnly`, and `TimeOnly`:
`{{.StartTime.Format timeFormats.RFC3339Nano}}`
