import unittest
import tempfile
import gzip

from .common import *

class CtStoreTests(unittest.TestCase):
    def test_cron_job(self):
        with tempfile.TemporaryDirectory() as d:
            command = ["/bin/sh", "-c", "echo Test"]
            job_id = "test/store/job"
            job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
            run(ct_store_path + [d], check=True, input=job_result.stdout)
            _, headers, _, _ = check_cronlog(self, job_result, exit="code 0", command=command, id=job_id)

            self.assertEqual(os.listdir(pathlib.Path(d)), ["test", "_invalid", "_err", "_incoming"])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_invalid"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_err"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_incoming"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"test"), ["store"])
            self.assertEqual(os.listdir(pathlib.Path(d)/"test"/"store"), ["job"])

            entries = os.listdir(pathlib.Path(d)/"test"/"store"/"job")
            self.assertEqual(len(entries), 1)
            self.assertEqual(
                entries[0].split(".")[0],
                datetime.datetime.fromisoformat(
                    re.sub("Z$", "+00:00", log_times(job_result.stdout)[0]),
                ).strftime("%Y-%m-%dT%H%M%SZ"),
            )
            self.assertEqual(entries[0].split(".")[-1], "gz")

            with gzip.open(pathlib.Path(d)/"test"/"store"/"job"/entries[0]) as fd:
                self.assertEqual(fd.read(), job_result.stdout)

    def test_cron_err(self):
        with tempfile.TemporaryDirectory() as d:
            command = ["/bin/sh", "-c", "echo Test"]
            job_result = run(ct_run_path + ["--"] + command)
            run(ct_store_path + [d], check=True, input=job_result.stdout)
            check_cronerr(self, job_result)

            self.assertEqual(os.listdir(pathlib.Path(d)), ["_invalid", "_err", "_incoming"])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_invalid"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_incoming"), [])

            entries = os.listdir(pathlib.Path(d)/"_err")
            self.assertEqual(len(entries), 1)

            with gzip.open(pathlib.Path(d)/"_err"/entries[0]) as fd:
                self.assertEqual(fd.read(), job_result.stdout)

    def test_invalid(self):
        with tempfile.TemporaryDirectory() as d:
            command = ["/bin/sh", "-c", "echo Test"]
            job_result = run(command)
            run(ct_store_path + [d], check=True, input=job_result.stdout)

            self.assertEqual(os.listdir(pathlib.Path(d)), ["_invalid", "_err", "_incoming"])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_err"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_incoming"), [])

            entries = os.listdir(pathlib.Path(d)/"_invalid")
            self.assertEqual(len(entries), 1)

            with gzip.open(pathlib.Path(d)/"_invalid"/entries[0]) as fd:
                self.assertEqual(fd.read(), job_result.stdout)

    def test_truncated(self):
        with tempfile.TemporaryDirectory() as d:
            command = ["/bin/sh", "-c", "echo Test"]
            job_result = run(ct_run_path + ["-I", "test-store-truncated", "--"] + command)
            run(ct_store_path + [d], check=True, input=job_result.stdout[:42])

            self.assertEqual(os.listdir(pathlib.Path(d)), ["_invalid", "_err", "_incoming"])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_err"), [])
            self.assertEqual(os.listdir(pathlib.Path(d)/"_incoming"), [])

            entries = os.listdir(pathlib.Path(d)/"_invalid")
            self.assertEqual(len(entries), 1)

            with gzip.open(pathlib.Path(d)/"_invalid"/entries[0]) as fd:
                self.assertEqual(fd.read(), job_result.stdout[:42])

    def test_multiple_repositories(self):
        with tempfile.TemporaryDirectory() as d1:
            with tempfile.TemporaryDirectory() as d2:
                command = ["/bin/sh", "-c", "echo Test"]
                job_id = "test/store/multi"
                job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
                run(ct_store_path + [d1, d2], check=True, input=job_result.stdout)

                for d in [d1, d2]:
                    entries = os.listdir(pathlib.Path(d)/"test"/"store"/"multi")
                    self.assertEqual(len(entries), 1)
                    with gzip.open(pathlib.Path(d)/"test"/"store"/"multi"/entries[0]) as fd:
                        self.assertEqual(fd.read(), job_result.stdout)

    def test_partial_failure(self):
        with tempfile.TemporaryDirectory() as d:
            command = ["/bin/sh", "-c", "echo Test"]
            job_id = "test/store/partial"
            job_result = run(ct_run_path + ["-I", job_id, "--"] + command)
            missing = pathlib.Path(d) / "missing"
            result = run(ct_store_path + [d, missing], input=job_result.stdout)
            self.assertNotEqual(result.returncode, 0)

            entries = os.listdir(pathlib.Path(d)/"test"/"store"/"partial")
            self.assertEqual(len(entries), 1)
            with gzip.open(pathlib.Path(d)/"test"/"store"/"partial"/entries[0]) as fd:
                self.assertEqual(fd.read(), job_result.stdout)
