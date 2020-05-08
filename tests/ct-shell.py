import unittest

from .common import *

class CtShellTests(unittest.TestCase):
    def test_cronjob(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-shell-job"
        job_result = run(ct_shell_path + ct_run_path + ["-I", job_id, "--"] + command)
        _, _, body, _ = check_cronlog(self, job_result, exit="code 0", command=command, id=job_id)

        self.assertEqual(len(body), 1)
        _, marker, data = body[0]
        self.assertEqual((marker, data), (b">", b"Test"))

    def test_cronerr(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_shell_path + ct_run_path + ["--"] + command)
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"CRONERR", joined_body)

    def test_invalid_stdout(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_result = run(ct_shell_path + command)
        check_cronerr(self, job_result)

    def test_stderr(self):
        command = ["/bin/sh", "-c", "echo Test >&2"]
        job_result = run(ct_shell_path + command)
        check_cronerr(self, job_result)

    def test_no_out(self):
        command = ["/bin/sh", "-c", "true"]
        job_result = run(ct_shell_path + command)
        check_cronerr(self, job_result)
