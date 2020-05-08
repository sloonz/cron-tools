# ct sign

## Synopsis

`**ct sign** [-U *user*] [-H *host*] -I *id* -k *signing-key* -- *command*`

## Overview

Generates a signed command intended to by used by `ct run`: only hosts
configured to accept the public key associated with the signed key will
allow the execution of the command. See the documentation of `ct run`
for more details on how to deploy this.

The signing key is in OpenSSH format. It is recommended to use ed25519
keys, even though rsa and ecdsa are supported.

Giving an user with the `-U` option or an hostname with the `-H` option
will cause `ct run` to only allow the execution of the command on the
given user and/or host. Not specifying those options will cause no
host/user limitation of the generated signed command.

Example:

```
$ ct sign -k /home/admin/cron_ed25519 -I myserver/root/backup -- /usr/local/bin/backup
eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJpZCI6Im15c2VydmVyL3Jvb3QvYmFja3VwIiwiQ29tbWFuZCI6WyIvdXNyL2xvY2FsL2Jpbi9iYWNrdXAiXX0.BnCip02Ia0NJ6kyW8wJD96dHD2QtKk-Cc2TXtOA0mglbkRs3a6OU2RdJS6wAGykqFa2_FeGyoMYuLWFlhXPQDA
$ ct run -k /home/admin/cron_ed25519.pub -s eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJpZCI6Im15c2VydmVyL3Jvb3QvYmFja3VwIiwiQ29tbWFuZCI6WyIvdXNyL2xvY2FsL2Jpbi9iYWNrdXAiXX0.BnCip02Ia0NJ6kyW8wJD96dHD2QtKk-Cc2TXtOA0mglbkRs3a6OU2RdJS6wAGykqFa2_FeGyoMYuLWFlhXPQDA
```
