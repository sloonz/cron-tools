# ct gen

## Synopsis

`**ct gen** [-U *user*] [-H *host*] [-k *signing-key*] -t *template* *cron-job-descriptions*

## Overview

`ct gen` generates a crontab based on a template and cron job
descriptions. See `ct-templates-format` for the documentation of the
template format, and `ct-jobs-format` for the documenation of the cron
jobs description format.

You can limit the crontab generation to a specific host (`-H) and/or
user (`-U`). The given template will be repeatedly to every cron job
description, and the result printed to standard output.

The following fields are provided to the template, mirroring the
information in the cron job description format.

* `.User`
* `.Host`
* `.ID`
* `.Command`
* `.Frequency`
* `.Location` (corresponding to the `tz` field)

In addition to the standard Go template functions and the standard
cron-tools template functions, `ct gen` also provides a `sign` function
to generate a signed command:

```
ssh user@host {{sign .}}
```

Using the `sign` function requires to provide the `-k` argument.

Example:

```
# Careful with that: will overwrite the crontab of the root user on myserver
ct gen -H myserver -t "{{.Frequency}} sudo -u {{shescape .User}} ct run -I {{shescape .ID}} {{shjoin .Command}}" crons-repository/_jobs/ | ssh root@myserver crontab -
```

The arguments of `ct gen` are files or directories containing cron
jobs descriptions. Directories are recursively scanned for `.json` or
`.jsonnet` files that are used to read descriptions. Descriptions files
are an array of JSON object description the cron jobs, for example:

```json
[
  {"user": "root", "host": "myserver", "id": "myserver/root/backup", "command": ["/usr/local/bin/backup"], "frequency": "@daily"},
  {"user": "root", "host": "otherserver", "id": "motherserver/root/backup", "command": ["/usr/local/bin/backup"], "frequency": "@daily"}
]
```

Example templates:

```
{{.Frequency}} ct shell ssh {{shescape .User}}@{{shescape .Host}} {{sign .}} | ct store | ct notify
{{.Frequency}} ct run -I {{shescape .ID}} -- {{shjoin .Command}} | ssh cronsrepo@storageserver
```
