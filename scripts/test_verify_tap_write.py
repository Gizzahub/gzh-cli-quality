"""Offline protocol/HTTP tests: never read a real credential or contact GitHub."""

import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import unittest
import urllib.error
from unittest import mock

spec = importlib.util.spec_from_file_location("verify_tap_write", Path(__file__).with_name("verify_tap_write.py"))
proof = importlib.util.module_from_spec(spec)
spec.loader.exec_module(proof)
SHA = "a" * 40
OTHER_SHA = "b" * 40
REF = proof.PREFIX + "123-1-" + "c" * 32
SECRET = "fake-credential-never-log-this"


def ref_reply(reference=REF, sha=SHA, status=200, kind="commit"):
    return status, json.dumps({"ref": reference, "object": {"type": kind, "sha": sha}}).encode()


ABSENT = (404, b'{"message":"Not Found"}')
MAIN = ref_reply("refs/heads/main")
CREATED = ref_reply(status=201)
PRESENT = ref_reply()
DELETED = (204, b"")


class FakeClient:
    def __init__(self, replies):
        self.replies = list(replies)
        self.calls = []
        self.phases = []

    def request(self, method, path, data=None, *, cleanup=False, phase="proof"):
        self.calls.append((method, path, data, cleanup))
        self.phases.append(phase)
        if not self.replies:
            raise AssertionError("unexpected extra request")
        reply = self.replies.pop(0)
        if isinstance(reply, Exception):
            raise reply
        return reply


class ProtocolTests(unittest.TestCase):
    def failure(self, replies, deletes=0):
        client = FakeClient(replies)
        with contextlib.redirect_stdout(io.StringIO()), self.assertRaises(proof.ProofError) as caught:
            proof.verify(client, REF)
        self.assertNotIn(SECRET, str(caught.exception))
        self.assertEqual(sum(call[0] == "DELETE" for call in client.calls), deletes)
        self.assertEqual(client.replies, [])
        return client

    def test_success_creates_only_ref_at_main_and_verifies_cleanup(self):
        client = FakeClient([MAIN, ABSENT, CREATED, PRESENT, PRESENT, DELETED, ABSENT])
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(proof.verify(client, REF), SHA)
        self.assertEqual([c[0] for c in client.calls], ["GET", "GET", "POST", "GET", "GET", "DELETE", "GET"])
        self.assertEqual(client.calls[2][1:3], ("git/refs", {"ref": REF, "sha": SHA}))
        self.assertEqual(client.calls[5][1], "git/refs/" + REF.removeprefix("refs/"))
        self.assertEqual(client.phases, ["main", "absence", "create", "verify", "cleanup-check", "cleanup-delete", "cleanup-absence"])
        self.assertTrue(all(call[3] for call in client.calls[-3:]))
        self.assertEqual(client.replies, [])

    def test_main_must_be_exact_commit_before_any_mutation(self):
        for reply in [(403, SECRET.encode()), (200, b"invalid"), ref_reply("refs/heads/other"), ref_reply("refs/heads/main", kind="tag"), ref_reply("refs/heads/main", sha="invalid")]:
            with self.subTest(reply=reply[0]):
                client = self.failure([reply])
                self.assertEqual([c[0] for c in client.calls], ["GET"])

    def test_preexisting_or_unproven_absence_is_never_deleted(self):
        for reply in [PRESENT, (404, b"<html>missing</html>"), (404, b'{"message":"proxy unavailable"}'), (403, SECRET.encode())]:
            with self.subTest(reply=reply[0]):
                client = self.failure([MAIN, reply])
                self.assertEqual([c[0] for c in client.calls], ["GET", "GET"])

    def test_uncertain_creation_only_reconciles_readonly_never_retries_or_deletes(self):
        for reply in [(422, b"{}"), (403, SECRET.encode()), RuntimeError(SECRET), (201, b"invalid"), ref_reply(sha=OTHER_SHA, status=201), ref_reply(reference="refs/heads/wrong", status=201), ref_reply(status=201, kind="tag"), ref_reply(status=200), (201, b'{"ref":"bad","ref":"bad"}')]:
            with self.subTest(reply=type(reply).__name__):
                client = self.failure([MAIN, ABSENT, reply, PRESENT])
                self.assertEqual([c[0] for c in client.calls], ["GET", "GET", "POST", "GET"])

    def test_owned_ref_mismatch_never_deletes_even_at_expected_sha(self):
        for reply in [ref_reply(sha=OTHER_SHA), ref_reply(reference="refs/heads/wrong"), ref_reply(kind="tag"), (200, b"invalid")]:
            with self.subTest(reply=reply[0]):
                self.failure([MAIN, ABSENT, CREATED, reply])

    def test_owned_verification_error_still_safely_cleans_but_fails(self):
        for reply in [RuntimeError(SECRET), (503, SECRET.encode())]:
            with self.subTest(reply=type(reply).__name__):
                self.failure([MAIN, ABSENT, CREATED, reply, PRESENT, DELETED, ABSENT], deletes=1)

    def test_cleanup_requires_matching_current_owned_ref(self):
        for reply in [ref_reply(sha=OTHER_SHA), (403, SECRET.encode()), (200, b"invalid"), RuntimeError(SECRET)]:
            with self.subTest(reply=type(reply).__name__):
                self.failure([MAIN, ABSENT, CREATED, PRESENT, reply])

    def test_cleanup_delete_errors_are_not_retried_and_always_fail_closed(self):
        for reply in [RuntimeError(SECRET), (403, SECRET.encode()), (204, b"unexpected body")]:
            with self.subTest(reply=type(reply).__name__):
                self.failure([MAIN, ABSENT, CREATED, PRESENT, PRESENT, reply, ABSENT], deletes=1)

    def test_cleanup_requires_canonical_final_404(self):
        for reply in [PRESENT, (403, SECRET.encode()), (404, b"<html>missing</html>"), (404, b'{"message":"proxy unavailable"}'), RuntimeError(SECRET)]:
            with self.subTest(reply=type(reply).__name__):
                self.failure([MAIN, ABSENT, CREATED, PRESENT, PRESENT, DELETED, reply], deletes=1)

    def test_json_schema_rejects_multiple_values_duplicates_and_invalid_utf8(self):
        for body in [b"{}\n{}", b'{"draft":true,"draft":false}', b"[]", b"null", b'{"x":NaN}', b'"\xff"']:
            with self.subTest(body=body):
                with self.assertRaises(proof.ProofError):
                    proof.object_body(body)

    def test_non_probe_namespace_never_reaches_transport(self):
        client = FakeClient([])
        with self.assertRaises(proof.ProofError):
            proof.verify(client, "refs/heads/main")
        self.assertEqual(client.calls, [])

    def test_readonly_reconciliation_does_not_claim_malformed_response_is_present(self):
        client = FakeClient([(200, b"invalid")])
        self.assertEqual(proof.reconcile(client, REF), "uncertain")
        self.assertEqual([call[0] for call in client.calls], ["GET"])


class Response(io.BytesIO):
    def __init__(self, status=200, body=b"{}", url=proof.API + "git/ref/heads/main"):
        super().__init__(body)
        self.code = status
        self.url = url

    def geturl(self):
        return self.url


class HTTPTests(unittest.TestCase):
    def test_fixed_origin_headers_and_request_timeout(self):
        opener = mock.Mock()
        opener.open.return_value = Response()
        client = proof.Client(SECRET, opener=opener, clock=lambda: 0)
        self.assertEqual(client.request("GET", "git/ref/heads/main"), (200, b"{}"))
        request = opener.open.call_args.args[0]
        self.assertEqual(request.full_url, proof.API + "git/ref/heads/main")
        self.assertEqual(request.get_header("Authorization"), "Bearer " + SECRET)
        self.assertEqual(request.get_header("X-github-api-version"), "2026-03-10")
        self.assertEqual(opener.open.call_args.kwargs["timeout"], proof.REQUEST_TIMEOUT)

    def test_main_write_and_uncontrolled_urls_are_rejected(self):
        opener = mock.Mock()
        client = proof.Client(SECRET, opener=opener, clock=lambda: 0)
        for method, path, data in [("DELETE", "git/refs/heads/main", None), ("PATCH", "git/refs/heads/main", {"sha": SHA}), ("GET", "https://attacker.invalid", None), ("GET", "git/ref/git/refs/" + REF.removeprefix("refs/"), None), ("POST", "git/refs", {"ref": "refs/heads/main", "sha": SHA}), ("POST", "git/refs", {"ref": None, "sha": SHA}), ("POST", "git/refs", {"ref": REF, "sha": SHA, "force": True})]:
            with self.assertRaises(proof.ProofError):
                client.request(method, path, data)
        opener.open.assert_not_called()

    def test_redirect_is_not_followed_and_changed_response_url_is_rejected(self):
        self.assertIsNone(proof.NoRedirect().redirect_request(None, None, 302, "redirect", {}, "https://attacker.invalid"))
        opener = mock.Mock()
        opener.open.return_value = Response(url="https://attacker.invalid")
        with self.assertRaises(proof.ProofError):
            proof.Client(SECRET, opener=opener).request("GET", "git/ref/heads/main")
        self.assertEqual(opener.open.call_count, 1)

    def test_timeout_and_transport_errors_do_not_expose_details(self):
        for cause in [TimeoutError(SECRET), RuntimeError(SECRET)]:
            opener = mock.Mock()
            opener.open.side_effect = cause
            with self.assertRaises(proof.ProofError) as caught:
                proof.Client(SECRET, opener=opener).request("GET", "git/ref/heads/main")
            self.assertNotIn(SECRET, str(caught.exception))

    def test_status_and_phase_are_visible_without_response_or_secret(self):
        opener = mock.Mock()
        opener.open.return_value = Response(status=403, body=json.dumps({"message": SECRET}).encode())
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            reply = proof.Client(SECRET, opener=opener).request("GET", "git/ref/heads/main", phase="main")
        self.assertEqual(reply[0], 403)
        self.assertIn("phase=main method=GET path=git/ref/heads/main HTTP=403", output.getvalue())
        self.assertNotIn(SECRET, output.getvalue())

    def test_real_http_error_shape_is_returned_without_logging_body(self):
        url = proof.API + "git/ref/heads/main"
        opener = mock.Mock()
        opener.open.side_effect = urllib.error.HTTPError(url, 403, SECRET, {}, io.BytesIO(SECRET.encode()))
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(proof.Client(SECRET, opener=opener).request("GET", "git/ref/heads/main", phase="main"), (403, SECRET.encode()))
        self.assertIn("HTTP=403", output.getvalue())
        self.assertNotIn(SECRET, output.getvalue())

    def test_total_deadline_reserves_cleanup_and_limits_response_size(self):
        now = [0]
        opener = mock.Mock()
        client = proof.Client(SECRET, opener=opener, clock=lambda: now[0])
        now[0] = 61
        with self.assertRaises(proof.ProofError):
            client.request("GET", "git/ref/heads/main")
        opener.open.assert_not_called()
        opener.open.return_value = Response()
        self.assertEqual(client.request("GET", "git/ref/heads/main", cleanup=True), (200, b"{}"))
        now[0] = 91
        with self.assertRaises(proof.ProofError):
            client.request("GET", "git/ref/heads/main", cleanup=True)
        opener.open.return_value = Response(body=b"x" * (proof.MAX_BODY + 1))
        with self.assertRaises(proof.ProofError):
            proof.Client(SECRET, opener=opener).request("GET", "git/ref/heads/main")

    def test_post_deadline_with_201_status_never_grants_ownership(self):
        now = [0]
        replies = [Response(body=MAIN[1]), Response(status=404, body=ABSENT[1], url=proof.API + proof.read_path(REF)), Response(status=201, body=CREATED[1], url=proof.API + "git/refs"), Response(body=PRESENT[1], url=proof.API + proof.read_path(REF))]
        calls = []

        def open_response(request, timeout):
            calls.append(request.get_method())
            if request.get_method() == "POST":
                now[0] = 61
            return replies.pop(0)

        opener = mock.Mock()
        opener.open.side_effect = open_response
        client = proof.Client(SECRET, opener=opener, clock=lambda: now[0])
        with contextlib.redirect_stdout(io.StringIO()), self.assertRaises(proof.ProofError):
            proof.verify(client, REF)
        self.assertEqual(calls, ["GET", "GET", "POST", "GET"])
        self.assertEqual(replies, [])


class EntryTests(unittest.TestCase):
    def test_cli_refuses_local_or_wrong_source_before_constructing_client(self):
        for env in [{}, {"GITHUB_ACTIONS": "true", "GITHUB_REPOSITORY": "other/repository"}]:
            with mock.patch.dict(os.environ, env, clear=True), mock.patch.object(proof, "Client") as client, contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(proof.main(), 1)
                client.assert_not_called()

    def test_actions_identity_credential_and_nonce(self):
        env = {"GITHUB_ACTIONS": "true", "GITHUB_REPOSITORY": proof.SOURCE, "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "1", "HOMEBREW_TAP_TOKEN": SECRET}
        output = io.StringIO()
        with mock.patch.dict(os.environ, env, clear=True), mock.patch.object(proof, "Client"), mock.patch.object(proof, "verify", return_value=SHA) as verify, mock.patch.object(proof.secrets, "token_hex", side_effect=["c" * 32, "d" * 32]), contextlib.redirect_stdout(output):
            self.assertEqual(proof.main(), 0)
            self.assertEqual(proof.main(), 0)
            self.assertNotEqual(verify.call_args_list[0].args[1], verify.call_args_list[1].args[1])
            self.assertTrue(all(proof.REF_PATTERN.fullmatch(call.args[1]) for call in verify.call_args_list))
        self.assertNotIn(SECRET, output.getvalue())
        for key, value in [("GITHUB_RUN_ID", "bad"), ("GITHUB_RUN_ATTEMPT", "0"), ("HOMEBREW_TAP_TOKEN", ""), ("HOMEBREW_TAP_TOKEN", "bad token")]:
            with mock.patch.dict(os.environ, dict(env, **{key: value}), clear=True), mock.patch.object(proof, "Client") as client, contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(proof.main(), 1)
                client.assert_not_called()


if __name__ == "__main__":
    unittest.main()
