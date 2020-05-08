# Logs format

There are two formats: the regular format and the failure format.

The failure format starts with the magic string `CRONERR\n`. The rest
of the file is unformatted text that should be presented as-is.

The regular format is a CRI-style text log with three streams:

* `stdout`
* `stderr`
* `meta`

Each line is:

```text
<timestamp> <stream> <flag> <message>
```

Where:

* `timestamp` is an RFC 3339 timestamp with nanoseconds
* `stream` is one of `stdout`, `stderr`, or `meta`
* `flag` is `F` for a complete line or `P` for a partial line
* `message` is plain text

The first regular log line is always a version marker:

```text
<timestamp> meta F cronlog-v0
```

This line is required and lets tools such as `ct shell` positively
identify a cron-tools log stream.

## Meta stream

The `meta` stream carries header/footer information as plain
`Key: Value` lines.

Header keys:

* `Command`: JSON array representing the executed command and its arguments
* `User`: Unix user running the command
* `Host`: machine name on which the command was run
* `Job-ID`: cron job identifier

Footer keys:

* `Exit`: either `code`, `signal`, or `other` followed by detailed information

Headers precede the `stdout`/`stderr` output records; footers follow them.

## Output streams

`stdout` and `stderr` carry the command output directly as plain text.
The `P` flag indicates a partial line, and `F` indicates that the
command output line ended with a newline.
