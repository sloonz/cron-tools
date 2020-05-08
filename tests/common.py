import pathlib
import subprocess
import os
import socket
import json
import datetime
import re
repo_path = (pathlib.Path(__file__).parent / "..").resolve()
tests_path = pathlib.Path(__file__).parent


def resolve_binary(name):
    return [repo_path / "ct", name]


ct_gen_path = resolve_binary("gen")
ct_run_path = resolve_binary("run")
ct_shell_path = resolve_binary("shell")
ct_sign_path = resolve_binary("sign")
ct_store_path = resolve_binary("store")
ct_notify_path = resolve_binary("notify")
ct_digest_path = resolve_binary("digest")

def run(*args, **kwargs):
    if "env" in kwargs:
        new_env = os.environ.copy()
        new_env.update(kwargs["env"])
        kwargs["env"] = new_env
    return subprocess.run(*args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, **kwargs)

def parse_ct_run_output(out):
    headers = {}
    body = []
    footers = {}
    log_type = None

    for l in out.splitlines():
        if log_type is None:
            if l == b"CRONERR":
                log_type = "CRONERR"
                continue
            else:
                log_type = "CRI"

        if log_type == "CRONERR":
            body.append(l)
        else:
            line = l.decode()
            timestamp, stream, flag, payload = line.split(" ", 3)
            if payload == "cronlog-v0":
                continue
            if stream == "meta":
                key, value = payload.split(": ", 1)
                if key == "Exit":
                    footers[key] = value
                else:
                    headers[key] = value
            else:
                marker = b"@>" if stream == "stdout" and flag == "P" else \
                    b"@!" if stream == "stderr" and flag == "P" else \
                    b">" if stream == "stdout" else b"!"
                body.append((timestamp.encode(), marker, payload.encode()))

    return log_type, headers, body, footers

def check_cronlog(t, job_result, command=None, id=None, exit_type=None, exit=None):
    log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)

    t.assertEqual(job_result.returncode, 0)
    t.assertEqual(job_result.stderr, b"")
    t.assertEqual(log_type, "CRI")
    t.assertEqual(headers["User"], os.getlogin())
    t.assertEqual(headers["Host"], socket.gethostname())
    t.assertEqual(set(headers.keys()), set(["User", "Host", "Job-ID", "Command"]))
    t.assertEqual(set(footers.keys()), set(["Exit"]))
    if command is not None:
        t.assertEqual(json.loads(headers["Command"]), command)
    if id is not None:
        t.assertEqual(headers["Job-ID"], id)
    if exit is not None:
        t.assertEqual(footers["Exit"], exit)
    if exit_type is not None:
        t.assertEqual(footers["Exit"].split(" ", 1)[0], exit_type)

    start_time, end_time = log_times(job_result.stdout)
    now = datetime.datetime.now(tz=datetime.timezone.utc)
    date_start = datetime.datetime.fromisoformat(re.sub("Z$", "+00:00", start_time))
    date_done = datetime.datetime.fromisoformat(re.sub("Z$", "+00:00", end_time))
    t.assertTrue(now >= date_start)
    t.assertTrue(now >= date_done)
    t.assertTrue(date_done >= date_start)
    t.assertTrue((now - date_start).total_seconds() < 60)
    t.assertTrue((now - date_done).total_seconds() < 60)

    return log_type, headers, body, footers

def check_cronerr(t, job_result):
    log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)

    t.assertNotEqual(job_result.returncode, 0)
    t.assertEqual(job_result.stderr, b"")
    t.assertEqual(log_type, "CRONERR")
    t.assertEqual(headers, {})
    t.assertEqual(footers, {})

    return log_type, headers, body, footers


def log_times(out):
    records = [line.decode().split(" ", 3) for line in out.splitlines()]
    start = records[0][0]
    end = next(ts for ts, stream, flag, payload in records
               if stream == "meta" and payload.startswith("Exit: "))
    return start, end
