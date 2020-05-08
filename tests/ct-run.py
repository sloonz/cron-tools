import unittest

from .common import *

class CtRunTests(unittest.TestCase):
    def test_missing_id(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_run_path + ["--"] + command)
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_missing_cmd(self):
        job_id = "test-run-missing-cmd"
        job_result = run(ct_run_path + ["-I", job_id])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_invalid_executable(self):
        command = ["/hopefully-non-existent-executable"]
        job_id = "test-run-invalid-exec"
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        check_cronlog(self, job_result, exit_type="other", command=command, id=job_id)

    def test_success(self):
        command = ["/bin/sh", "-c", "echo Test 1; echo Test 2 >&2; sleep 0.2; echo Test 3 >&2; echo Test 4"]
        job_id = "test-run-success"
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        _, _, body, _ = check_cronlog(self, job_result, exit="code 0", command=command, id=job_id)

        self.assertEqual(len(body), 4)
        times = [
            datetime.datetime.fromisoformat(re.sub("Z$", "+00:00", ts.decode()))
            for ts, _, _ in body
        ]
        entries = [(marker, data) for _, marker, data in body]
        d = min(times[2:]) - max(times[:2])

        self.assertTrue(d.total_seconds() >= 0.2)
        self.assertEqual(set(entries[:2]), set([(b">", b"Test 1"), (b"!", b"Test 2")]))
        self.assertEqual(set(entries[2:]), set([(b">", b"Test 4"), (b"!", b"Test 3")]))

    def test_exit_code(self):
        command = ["/bin/sh", "-c", "exit 3"]
        job_id = "test-run-exit-code"
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        check_cronlog(self, job_result, exit="code 3", command=command, id=job_id)

    def test_signal(self):
        command = ["/bin/sh", "-c", "kill $$"]
        job_id = "test-run-signal-1"
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        check_cronlog(self, job_result, exit="signal terminated", command=command, id=job_id)

        command = ["/bin/sh", "-c", "kill -KILL $$"]
        job_id = "test-run-signal-2"
        job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
        check_cronlog(self, job_result, exit="signal killed", command=command, id=job_id)
