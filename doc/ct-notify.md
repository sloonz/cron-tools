# ct notify

## Synopsis

`**ct notify** [-V] [-f *template-file*] [-p *template-path*]`

`**ct --debug notify** [-V] [-f *template-file*] [-p *template-path*]`

## Overview

`ct notify` reads a cron log and applies templates to them, as given by
the `-f` command line argument (or `$HOME/.config/cron-tools/notify.gotpl`
or `/etc/cron-tools/notify.gotpl` if none is given on the
command line) for a single template file and the `-p` command
line argument (or `$HOME/.config/cron-tools/notify.d/**/*.gotpl` or
`/etc/cron-tools/notify.d/**/*.gotpl` if none is given on the command line)
for a directory containing template.

Templates are Go templates (see `ct-template-format` for more information)
split into subtemplates. The main subtemplate is the `action` subtemplate,
which dictates what `ct notify` should actually do with the rest of the
template. There is currently two possible actions :

* `pipe` will evaluate the `data` subtemplate and outputs its result on
`ct notify` standard output.

* `exec` will evaluate the `command` subtemplate, the optional `stdin`
subtemplate, will execute the given command, passing it the given standard
input (or none if none is given). If the command fails, this will cause
`ct notify` to terminate with an error.

An empty action means that the template should be skipped. An unkwown
action will cause an error.

When invoked as `ct --debug notify`, `ct notify` logs the evaluated
`action` template and, depending on it, the evaluated `data`,
`command`, and `stdin` templates to standard error.

The following variables are available in the templates :

* `Format`, a string which corresponds on the type of processed log. Can
be `LOG`, `ERR` or `INVALID`.

* `StartTime` and `EndTime`, Go `time.Time` values from the `cronlog-v0` marker and
`Exit` record timestamps. A missing event leaves the corresponding value zero.

* `Headers`, a string-to-string dictionary which corresponds to the
headers of the log.

* `Footers`, a string-to-string dictionary which corresponds to the
headers of the log.

* RawData` is a string which corresponds to the raw body of the log
(including unparsed headers and footers.

* For `LOG` logs, `Lines` is an array of command output. Each line is
an object containing the following fields :

    - `TimeOffset`: a number describing the number of seconds elapsed
    between the launch of the command and the time where the line
    was emitted

    - `IsStderr`: true if the line was emitted on stderr, false if it
    was emitted on stdout

    - `Data`: line text

    - `IsChunk`: true if the line line was not a real line (was not
    terminated by an end-of-line character) but instead a single line
    split into multiple lines
