import unittest
import tempfile
import struct

from .common import *

NOTIFY_JSON = b"""
{{define "action"}}pipe{{end}}
{{define "data"}}{{toJson .}}{{end}}
"""

NOTIFY_MAIL = b"""
{{define "action"}}{{if or (ne .Format "LOG") (ne (get .Footers "Exit") "code 0")}}exec{{end}}{{end}}
{{define "subject"}}{{if ne .Format "LOG"}}Unknown cron failure on {{hostname}}{{else}}{{get .Headers "Job-ID"}}: {{get .Footers "Exit"}}{{end}}{{end}}
{{define "command"}}jq -Rs '{stdin: ., args: $ARGS.positional}' --args -- mail -s {{shescape (eval "subject" .)}} test@example.com{{end}}
{{define "stdin"}}{{if eq .Format "LOG"}}{{template "stdin-log" .}}{{else}}{{template "stdin-err" .}}{{end}}{{end}}
{{define "stdin-log" -}}
User: {{get .Headers "User"}}@{{get .Headers "Host"}}
Command: {{shjoin (fromJson (get .Headers "Command"))}}
Date: {{.StartTime.Format timeFormats.RFC3339Nano}}

{{range .Lines -}}
{{.TimeOffset}}{{if .IsStderr}}!{{else}}>{{end}} {{.Data}}{{if not .IsChunk}}
{{end}}
{{- end}}{{end}}
{{define "stdin-err"}}{{.RawData}}{{end}}
"""

NOTIFY_SLACK = b"""
{{define "action"}}{{if or (ne .Format "LOG") (ne (get .Footers "Exit") "code 0")}}exec{{end}}{{end}}
{{define "command"}}jq '{stdin: ., args: $ARGS.positional}' --args -- curl -s -X POST -H 'Content-type: application/json' -d @- https://hooks.slack.com/services/{{env "SLACK_SECRET"}}{{end}}
{{define "stdin-err" -}}{
  "blocks": [
    {
      "type": "header",
      "text": {"type": "plain_text", "emoji": false, "text": "Cron Job Failure on {{hostname}}"}
    }
  ]
}{{- end}}
{{define "stdin-log" -}}{
  "blocks": [
    {
      "type": "header",
      "text": {"type": "plain_text", "emoji": false, "text": "Cron Job Failure"}
    },
    {
      "type": "section",
      "text": {"type": "mrkdwn", "text": {{toJson (print "*Command:* `" (shjoin (fromJson (get .Headers "Command"))) "`")}}}
    },
    {
      "type": "section",
      "fields": [
        {"type": "mrkdwn", "text": {{toJson (print "*Job ID:* " (get .Headers "Job-ID"))}}},
        {"type": "mrkdwn", "text": {{toJson (print "*User:* " (get .Headers "User"))}}},
        {"type": "mrkdwn", "text": {{toJson (print "*Date:* " (.StartTime.Format timeFormats.RFC3339Nano))}}},
        {"type": "mrkdwn", "text": {{toJson (print "*Host:* " (get .Headers "Host"))}}},
        {"type": "mrkdwn", "text": {{toJson (print "*Exit:* " (get .Footers "Exit"))}}}
      ]
    }
  ]
}{{- end}}
{{define "stdin"}}{{if eq .Format "LOG"}}{{template "stdin-log" .}}{{else}}{{template "stdin-err" .}}{{end}}{{end}}
"""

EXPECTED_MAIL_BODY = """
User: {user}@{host}
Command: {cmd}
Date: {start}

{offset}> Test
""".lstrip()

def offset_from_entry(start_time, entry):
    ts, _, _ = entry
    return (
        datetime.datetime.fromisoformat(re.sub("Z$", "+00:00", ts.decode())) -
        datetime.datetime.fromisoformat(re.sub("Z$", "+00:00", start_time))
    ).total_seconds()

def float32(value):
    return struct.unpack("!f", struct.pack("!f", value))[0]

def offset32_from_entry(start_time, entry):
    return float32(offset_from_entry(start_time, entry))

def assert_single_output_line(testcase, notif, start_time, body):
    testcase.assertEqual(len(notif["Lines"]), 1)
    line = notif["Lines"][0]
    testcase.assertEqual(line["IsChunk"], False)
    testcase.assertEqual(line["IsStderr"], False)
    testcase.assertEqual(line["Data"], "Test")
    testcase.assertAlmostEqual(line["TimeOffset"], offset32_from_entry(start_time, body[0]), places=5)

class CtNotifyTests(unittest.TestCase):
    def test_debug_exec_templates(self):
        job_id = "test-notify-debug"
        command = ["/bin/sh", "-c", "echo Test; exit 1"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_MAIL)
            tpl_file.flush()

            notify_result = run([repo_path / "ct", "--debug", "notify", "-f", tpl_file.name], input=job_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            stderr = notify_result.stderr.decode()
            self.assertIn("_main: action:\nexec\n", stderr)
            self.assertIn("_main: command:\n", stderr)
            self.assertIn("mail -s 'test-notify-debug: code 1' test@example.com", stderr)
            self.assertIn("_main: stdin:\nUser: ", stderr)

    def test_json_invalid(self):
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_JSON)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=b"Invalid log")
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "StartTime": "0001-01-01T00:00:00Z",
                "EndTime": "0001-01-01T00:00:00Z",
                "Format": "INVALID",
                "Headers": {},
                "Footers": {},
                "Lines": [],
                "RawData": "Invalid log",
            })

    def test_json_err(self):
        job_result = run(ct_run_path + ["--", "/bin/sh", "-c", "echo Test"])
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_JSON)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "StartTime": "0001-01-01T00:00:00Z",
                "EndTime": "0001-01-01T00:00:00Z",
                "Format": "ERR",
                "Headers": {},
                "Footers": {},
                "Lines": [],
                "RawData": job_result.stdout.decode(),
            })

    def test_json_fail(self):
        job_id = "test-notify-json-fail"
        command = ["/bin/sh", "-c", "echo Test; exit 1"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)
        start_time, end_time = log_times(job_result.stdout)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_JSON)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            notif = json.loads(notify_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(set(notif.keys()), set(["StartTime", "EndTime", "Format", "Headers", "Footers", "Lines", "RawData"]))
            self.assertEqual(notif["Format"], "LOG")
            self.assertEqual(notif["StartTime"], start_time)
            self.assertEqual(notif["EndTime"], end_time)
            self.assertEqual(notif["RawData"], job_result.stdout.decode())
            self.assertEqual(notif["Headers"], headers)
            self.assertEqual(notif["Footers"], footers)
            assert_single_output_line(self, notif, start_time, body)

    def test_json_ok(self):
        job_id = "test-notify-json-ok"
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)
        start_time, end_time = log_times(job_result.stdout)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_JSON)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            notif = json.loads(notify_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(set(notif.keys()), set(["StartTime", "EndTime", "Format", "Headers", "Footers", "Lines", "RawData"]))
            self.assertEqual(notif["Format"], "LOG")
            self.assertEqual(notif["StartTime"], start_time)
            self.assertEqual(notif["EndTime"], end_time)
            self.assertEqual(notif["RawData"], job_result.stdout.decode())
            self.assertEqual(notif["Headers"], headers)
            self.assertEqual(notif["Footers"], footers)
            assert_single_output_line(self, notif, start_time, body)

    def test_mail_invalid(self):
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_MAIL)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=b"Invalid log")
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "args": ["mail", "-s", "Unknown cron failure on %s" % socket.gethostname(), "test@example.com"],
                "stdin": "Invalid log",
            })

    def test_mail_err(self):
        job_result = run(ct_run_path + ["--", "/bin/sh", "-c", "echo Test"])
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_MAIL)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            notif = json.loads(notify_result.stdout)
            self.assertEqual(notif["args"], ["mail", "-s", "Unknown cron failure on %s" % socket.gethostname(), "test@example.com"])
            self.assertEqual(notif["stdin"][:8], "CRONERR\n")

    def test_mail_fail(self):
        job_id = "test-notify-mail-fail"
        command = ["/bin/sh", "-c", "echo Test; exit 1"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)
        start_time, end_time = log_times(job_result.stdout)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_MAIL)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            notif = json.loads(notify_result.stdout)
            self.assertEqual(notif["args"], ["mail", "-s", "test-notify-mail-fail: code 1", "test@example.com"])
            lines = notif["stdin"].splitlines()
            self.assertEqual("\n".join(lines[:4]) + "\n", EXPECTED_MAIL_BODY.format(
                user=os.getlogin(),
                host=socket.gethostname(),
                start=start_time,
                cmd="/bin/sh -c 'echo Test; exit 1'",
                offset="{offset}",
            ).split("{offset}", 1)[0])
            offset, rest = lines[4].split(">", 1)
            self.assertAlmostEqual(float(offset), offset32_from_entry(start_time, body[0]), places=5)
            self.assertEqual(rest, " Test")

    def test_mail_ok(self):
        job_id = "test-notify-mail-ok"
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_MAIL)
            tpl_file.flush()
            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout)
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")
            self.assertEqual(notify_result.stdout, b"")

    def test_slack_invalid(self):
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_SLACK)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=b"Invalid log", env={"SLACK_SECRET": "VV3cTP5OqIHMLK8RjyH62g"})
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "args": ["curl", "-s", "-X", "POST", "-H", "Content-type: application/json", "-d", "@-", "https://hooks.slack.com/services/VV3cTP5OqIHMLK8RjyH62g"],
                "stdin": {
                    "blocks": [{
                        "type": "header",
                        "text": {"type": "plain_text", "emoji": False, "text": "Cron Job Failure on %s" % socket.gethostname()}
                    }]
                }
            })

    def test_slack_err(self):
        job_result = run(ct_run_path + ["--", "/bin/sh", "-c", "echo Test"])
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_SLACK)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout, env={"SLACK_SECRET": "VV3cTP5OqIHMLK8RjyH62g"})
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "args": ["curl", "-s", "-X", "POST", "-H", "Content-type: application/json", "-d", "@-", "https://hooks.slack.com/services/VV3cTP5OqIHMLK8RjyH62g"],
                "stdin": {
                    "blocks": [{
                        "type": "header",
                        "text": {"type": "plain_text", "emoji": False, "text": "Cron Job Failure on %s" % socket.gethostname()}
                    }]
                }
            })

    def test_slack_fail(self):
        job_id = "test-notify-slack-fail"
        command = ["/bin/sh", "-c", "echo Test; exit 1"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        log_type, headers, body, footers = parse_ct_run_output(job_result.stdout)
        start_time, end_time = log_times(job_result.stdout)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_SLACK)
            tpl_file.flush()

            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout, env={"SLACK_SECRET": "VV3cTP5OqIHMLK8RjyH62g"})
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")

            self.assertEqual(json.loads(notify_result.stdout), {
                "args": ["curl", "-s", "-X", "POST", "-H", "Content-type: application/json", "-d", "@-", "https://hooks.slack.com/services/VV3cTP5OqIHMLK8RjyH62g"],
                "stdin": {
                    "blocks": [
                        {
                            "type": "header",
                            "text": {"type": "plain_text", "emoji": False, "text": "Cron Job Failure"}
                        },
                        {
                            "type": "section",
                            "text": {"type": "mrkdwn", "text": "*Command:* `/bin/sh -c 'echo Test; exit 1'`"}
                        },
                        {
                            "type": "section",
                            "fields": [
                                {"type": "mrkdwn", "text": "*Job ID:* test-notify-slack-fail"},
                                {"type": "mrkdwn", "text": "*User:* %s" % os.getlogin()},
                                {"type": "mrkdwn", "text": "*Date:* %s" % start_time},
                                {"type": "mrkdwn", "text": "*Host:* %s" % socket.gethostname()},
                                {"type": "mrkdwn", "text": "*Exit:* code 1"},
                            ]
                        }
                    ]
                }
            })

    def test_slack_ok(self):
        job_id = "test-notify-slack-ok"
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        with tempfile.NamedTemporaryFile() as tpl_file:
            tpl_file.write(NOTIFY_SLACK)
            tpl_file.flush()
            notify_result = run(ct_notify_path + ["-f", tpl_file.name], input=job_result.stdout, env={"SLACK_SECRET": "VV3cTP5OqIHMLK8RjyH62g"})
            self.assertEqual(notify_result.returncode, 0)
            self.assertEqual(notify_result.stderr, b"")
            self.assertEqual(notify_result.stdout, b"")
