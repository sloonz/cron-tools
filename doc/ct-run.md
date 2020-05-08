# ct run

## Synopsis

```
**ct run** -I *id* -- *command*
**ct run** [-k *authorized-keys-file*] *signed-command*
SSH_ORIGINAL_COMMAND=*signed-command* **ct run** [-k *authorized-keys-file*]
```

## Description

`ct run` executes a command and outputs an associated cron log, which
contains the standard output of the command, the standard error of the
command and additional information like the user running it, or its
exit status. The emitted log is expected to be processed by tools like
`ct notify` or `ct store`.

There is two modes of operation: signed and unsigned.

In unsigned mode, you simply specify the command and the cron job
identifier, and `ct run` will execute it :

```
ct run -I myserver/root/backup /usr/local/bin/backup
```

In signed mode, you specify a blob containing the command and the cron
job identifier. The blob is cryptographically signed, and the command
will be executed only if the signature matches a public key in the
authorized keys file :

```
ct run -k /etc/cron-tool/authorized-keys -s eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJpZCI6Im15c2VydmVyL3Jvb3QvYmFja3VwIiwiQ29tbWFuZCI6WyIvdXNyL2xvY2FsL2Jpbi9iYWNrdXAiXX0.BnCip02Ia0NJ6kyW8wJD96dHD2QtKk-Cc2TXtOA0mglbkRs3a6OU2RdJS6wAGykqFa2_FeGyoMYuLWFlhXPQDA
```

## Usage with SSH

The signed command can also be read from the `SSH_ORIGINAL_COMMAND`
environment variable, which allows you to put something like that in your
`~/.ssh/authorized_keys` file :

```
command="/usr/bin/ct run -k /etc/cron-tools/authorized-keys" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJnuyYa0lNqAXGVLNdWAtWLNShmmOsFK9XpMJuJUjh9e
```

Then the owner of the `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJnuyYa0lNqAXGVLNdWAtWLNShmmOsFK9XpMJuJUjh9e`
key will only be allowed to run commands through `ct run`, by running ssh that way:

```
ssh root@myserver eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJpZCI6Im15c2VydmVyL3Jvb3QvYmFja3VwIiwiQ29tbWFuZCI6WyIvdXNyL2xvY2FsL2Jpbi9iYWNrdXAiXX0.BnCip02Ia0NJ6kyW8wJD96dHD2QtKk-Cc2TXtOA0mglbkRs3a6OU2RdJS6wAGykqFa2_FeGyoMYuLWFlhXPQDA
```

(notice that no command is provided, only directly the signed command)

A command can be signed with `ct sign`.

## The authorized keys file

Contains a list of public keys used to authorize signed commands. This
is similar to the `~/.ssh/authorized-keys` (one key per line, key format
is the same than OpenSSH).

The predefined following paths are read if existing, in addition with the
one provided as an argument:

* `/etc/cron-tools/authorized-keys`
* `$HOME/.config/cron-tools/authorized-keys`
