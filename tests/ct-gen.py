import unittest

from .common import *

class CtGenTests(unittest.TestCase):
    def test_simple_template(self):
        template = "{{.Frequency}} ct run -I {{shescape .ID}} -- {{shjoin .Command}} | ssh cronsrepo@storageserver"
        result = run(ct_gen_path + ["-t", template, tests_path/"jobs"])
        self.assertEqual(result.stderr, b"")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(set(result.stdout.splitlines()), set([
            b"# mainserver/root/backup",
            b"@daily ct run -I mainserver/root/backup -- /usr/local/bin/backup | ssh cronsrepo@storageserver",
            b"# mainserver/www/acme certs",
            b"2 4 * * * ct run -I 'mainserver/www/acme certs' -- /usr/local/bin/acme-certs 'renew certificates' | ssh cronsrepo@storageserver",
            b"# mainserver/www/logrotate",
            b"@daily ct run -I mainserver/www/logrotate -- /usr/bin/logrotate /home/www/conf/logrotate.conf | ssh cronsrepo@storageserver",
            b"# otherserver/root/backup",
            b"@daily ct run -I otherserver/root/backup -- /usr/local/bin/backup | ssh cronsrepo@storageserver",
        ]))

    def test_filter_host(self):
        template = "{{.Frequency}} ct run -I {{shescape .ID}} -- {{shjoin .Command}} | ssh cronsrepo@storageserver"
        result = run(ct_gen_path + ["-H", "mainserver", "-t", template, tests_path/"jobs"])
        self.assertEqual(result.stderr, b"")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(set(result.stdout.splitlines()), set([
            b"# mainserver/root/backup",
            b"@daily ct run -I mainserver/root/backup -- /usr/local/bin/backup | ssh cronsrepo@storageserver",
            b"# mainserver/www/acme certs",
            b"2 4 * * * ct run -I 'mainserver/www/acme certs' -- /usr/local/bin/acme-certs 'renew certificates' | ssh cronsrepo@storageserver",
            b"# mainserver/www/logrotate",
            b"@daily ct run -I mainserver/www/logrotate -- /usr/bin/logrotate /home/www/conf/logrotate.conf | ssh cronsrepo@storageserver",
        ]))

    def test_filter_user(self):
        template = "{{.Frequency}} ct run -I {{shescape .ID}} -- {{shjoin .Command}} | ssh cronsrepo@storageserver"
        result = run(ct_gen_path + ["-U", "root", "-t", template, tests_path/"jobs"])
        self.assertEqual(result.stderr, b"")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(set(result.stdout.splitlines()), set([
            b"# mainserver/root/backup",
            b"@daily ct run -I mainserver/root/backup -- /usr/local/bin/backup | ssh cronsrepo@storageserver",
            b"# otherserver/root/backup",
            b"@daily ct run -I otherserver/root/backup -- /usr/local/bin/backup | ssh cronsrepo@storageserver",
        ]))

    def test_filter_hostuser(self):
        template = "{{.Frequency}} ct run -I {{shescape .ID}} -- {{shjoin .Command}} | ssh cronsrepo@storageserver"
        result = run(ct_gen_path + ["-U", "www", "-H", "mainserver", "-t", template, tests_path/"jobs"])
        self.assertEqual(result.stderr, b"")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(set(result.stdout.splitlines()), set([
            b"# mainserver/www/acme certs",
            b"2 4 * * * ct run -I 'mainserver/www/acme certs' -- /usr/local/bin/acme-certs 'renew certificates' | ssh cronsrepo@storageserver",
            b"# mainserver/www/logrotate",
            b"@daily ct run -I mainserver/www/logrotate -- /usr/bin/logrotate /home/www/conf/logrotate.conf | ssh cronsrepo@storageserver",
        ]))

    def test_sign(self):
        template = "{{.Frequency}} ct shell ssh {{shescape .User}}@{{shescape .Host}} {{sign .}} | ct store | ct notify"
        result = run(ct_gen_path + ["-t", template, "-k", tests_path/"key_ed25519", tests_path/"jobs"])
        self.assertEqual(result.stderr, b"")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(set(result.stdout.splitlines()), set([
            b"# mainserver/root/backup",
            b"@daily ct shell ssh root@mainserver eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoicm9vdCIsImhvc3QiOiJtYWluc2VydmVyIiwiaWQiOiJtYWluc2VydmVyL3Jvb3QvYmFja3VwIiwiY29tbWFuZCI6WyIvdXNyL2xvY2FsL2Jpbi9iYWNrdXAiXX0.Te7tNNYRMRDjz_1DI6oaV-nWitXEiJc-DaLKdiiv_FtGx0xM7tSAgx4ASl0wnrgO92vN_7g6NQbKU01IE5aUCw | ct store | ct notify",
            b"# mainserver/www/acme certs",
            b"2 4 * * * ct shell ssh www@mainserver eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoid3d3IiwiaG9zdCI6Im1haW5zZXJ2ZXIiLCJpZCI6Im1haW5zZXJ2ZXIvd3d3L2FjbWUgY2VydHMiLCJjb21tYW5kIjpbIi91c3IvbG9jYWwvYmluL2FjbWUtY2VydHMiLCJyZW5ldyBjZXJ0aWZpY2F0ZXMiXX0._t3cGi7uAmUyzXOdGTlZaTQNX7W64RjKOG4YVEfFz0aD8bCtlAud4an7jCX_iBA4MtTb_fmH1siIHH0J15VkBQ | ct store | ct notify",
            b"# mainserver/www/logrotate",
            b"@daily ct shell ssh www@mainserver eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoid3d3IiwiaG9zdCI6Im1haW5zZXJ2ZXIiLCJpZCI6Im1haW5zZXJ2ZXIvd3d3L2xvZ3JvdGF0ZSIsImNvbW1hbmQiOlsiL3Vzci9iaW4vbG9ncm90YXRlIiwiL2hvbWUvd3d3L2NvbmYvbG9ncm90YXRlLmNvbmYiXX0.ATcogLjiQrZsfE9TdsNk4ytH6s4pteLqSS8-DwLandtj-W5XnhHMhPZec18IqfNkxqSYDCW4UdMCAexdZo01CQ | ct store | ct notify",
            b"# otherserver/root/backup",
            b"@daily ct shell ssh root@otherserver eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoicm9vdCIsImhvc3QiOiJvdGhlcnNlcnZlciIsImlkIjoib3RoZXJzZXJ2ZXIvcm9vdC9iYWNrdXAiLCJjb21tYW5kIjpbIi91c3IvbG9jYWwvYmluL2JhY2t1cCJdfQ.PU5yUJC5aeSgRfnFg9RqSU_fETkZ7noMIF5H3javhB_DRwXXqFmrc5bj2vDXLqsB1Y0jr2cslHI2FfsYHgUXAw | ct store | ct notify",
        ]))
