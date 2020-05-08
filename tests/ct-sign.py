import unittest
import base64
import os
import socket

from .common import *

def tamper_jwt_payloat(signed_job):
    header, payload, sig = signed_job.split(b".")
    # Seriously python ? 2021 and still choking on the JWT variant of base64 ? Am I missing something ?
    # See: https://bugs.python.org/issue29427
    padding = b"=" * ((4-(len(payload)%4))%4)
    tampered_payload = base64.urlsafe_b64encode(base64.urlsafe_b64decode(payload + padding).lower()).rstrip(b"=")
    return b"%s.%s.%s" % (header, tampered_payload, sig)

class SignTests(unittest.TestCase):
    def assert_single_stdout_test(self, body):
        self.assertEqual(len(body), 1)
        _, marker, data = body[0]
        self.assertEqual((marker, data), (b">", b"Test"))

    def test_rsa_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-rsa-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_rsa", "--"] + command).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_rsa.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_ecdsa_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-ecdsa-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ecdsa", "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ecdsa.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_ed25519_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-ed25519-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_rsa_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-rsa-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_rsa", "--"] + command).stdout.strip()
        tampered_signed_job = tamper_jwt_payloat(signed_job)

        job_result = run(ct_run_path + ["-k", tests_path/"key_rsa.pub", "-s", tampered_signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"test", joined_body)

    def test_ecdsa_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-ecdsa-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ecdsa", "--"] + command, check=True).stdout.strip()
        tampered_signed_job = tamper_jwt_payloat(signed_job)

        job_result = run(ct_run_path + ["-k", tests_path/"key_ecdsa.pub", "-s", tampered_signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"test", joined_body)

    def test_ed25519_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-ed25519-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "--"] + command, check=True).stdout.strip()
        tampered_signed_job = tamper_jwt_payloat(signed_job)

        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", tampered_signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"test", joined_body)

    def test_authorized_keys_ok(self):
        # tests/key_ed25519.pub is in the authorized_keys file
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-authorized-keys-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"authorized_keys", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_authorized_keys_fail(self):
        # tests/key_rsa.pub is  NOT in the authorized_keys file
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-authorized-keys-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_rsa", "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"authorized_keys", "-s", signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_ssh_original_command(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-ssh-original-command"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"authorized_keys"], env={"SSH_ORIGINAL_COMMAND": signed_job})
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_sign_user_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-user-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-U", os.getlogin(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_sign_user_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-user-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-U", "invalid-%s" % os.getlogin(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_sign_host_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-host-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-H", socket.gethostname(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_sign_host_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-host-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-H", "invalid-%s" % socket.gethostname(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_sign_userhost_ok(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-userhost-ok"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-U", os.getlogin(), "-H", socket.gethostname(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronlog(self, job_result, exit="code 0", id=job_id, command=command)

        self.assert_single_stdout_test(body)

    def test_sign_userhost_user_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-userhost-user-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-U", "invalid-%s" % os.getlogin(), "-H", socket.gethostname(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)

    def test_sign_userhost_host_fail(self):
        command = ["/bin/sh", "-c", "echo Test"]
        job_id = "test-sign-userhost-host-fail"
        signed_job = run(ct_sign_path + ["-I", job_id, "-k", tests_path/"key_ed25519", "-U", os.getlogin(), "-H", "invalid-%s" % socket.gethostname(), "--"] + command, check=True).stdout.strip()
        job_result = run(ct_run_path + ["-k", tests_path/"key_ed25519.pub", "-s", signed_job])
        log_type, headers, body, footers = check_cronerr(self, job_result)

        joined_body = b"\n".join(body)
        self.assertNotIn(b"Test", joined_body)
