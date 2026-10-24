"""Prove the Actions tap credential can create/delete only a unique temporary ref.

REST ref inspection/deletion is not atomic: this requires exclusive ownership of
the nonce namespace. A changed or uncertain ref is left for operator inspection.
These refs are operational credential probes, not authoring task branches or CE
runs: they point only to an existing main commit and receive no files/commits.
"""

import json
import os
import re
import secrets
import sys
import time
import urllib.error
import urllib.request

SOURCE = "Gizzahub/gzh-cli-quality"
REPOSITORY = "Gizzahub/homebrew-tap"
API = "https://api.github.com/repos/" + REPOSITORY + "/"
API_VERSION = "2026-03-10"
PREFIX = "refs/heads/ce-tap-token-proof/"
REF_PATTERN = re.compile(re.escape(PREFIX) + r"[1-9][0-9]{0,19}-[1-9][0-9]{0,5}-[0-9a-f]{32}")
SHA_PATTERN = re.compile(r"[0-9a-f]{40}")
REQUEST_TIMEOUT = 10
TOTAL_TIMEOUT = 90
CLEANUP_RESERVE = 30
MAX_BODY = 65536
PHASES = {"proof", "main", "absence", "create", "verify", "reconcile", "cleanup-check", "cleanup-delete", "cleanup-absence"}


class ProofError(Exception):
    """Only controlled, credential-free messages may escape the protocol."""


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, url):
        return None


def object_body(body):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result

    def invalid_constant(value):
        raise ValueError("invalid JSON constant")

    try:
        value = json.loads(body.decode("utf-8"), object_pairs_hook=pairs, parse_constant=invalid_constant)
        if not isinstance(value, dict):
            raise ValueError("not an object")
        return value
    except Exception:
        raise ProofError("GitHub API returned an invalid JSON object") from None


def absent(reply):
    status, body = reply
    if status != 404:
        return False
    try:
        return object_body(body).get("message") == "Not Found"
    except ProofError:
        return False


def matching(reply, reference, sha, status=200):
    if reply[0] != status:
        return False
    try:
        value = object_body(reply[1])
        obj = value.get("object")
        return (value.get("ref") == reference and isinstance(obj, dict)
                and obj.get("type") == "commit" and obj.get("sha") == sha)
    except ProofError:
        return False


class Client:
    def __init__(self, token, *, opener=None, clock=time.monotonic):
        self._token = token
        self._clock = clock
        self._deadline = clock() + TOTAL_TIMEOUT
        self._opener = opener if opener is not None else urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    def _remaining(self, cleanup):
        remaining = self._deadline - self._clock() - (0 if cleanup else CLEANUP_RESERVE)
        if remaining <= 0:
            raise ProofError("GitHub API proof deadline exceeded")
        return remaining

    def request(self, method, path, data=None, *, cleanup=False, phase="proof"):
        # No URLs supplied by the API response, environment or command line.
        probe = path.removeprefix("git/ref/") if path.startswith("git/ref/") else path.removeprefix("git/refs/")
        probe_valid = REF_PATTERN.fullmatch("refs/" + probe) is not None
        allowed = ((method == "GET" and path == "git/ref/heads/main")
                   or (method == "GET" and path.startswith("git/ref/") and probe_valid)
                   or (method == "DELETE" and path.startswith("git/refs/") and probe_valid)
                   or (method == "POST" and path == "git/refs" and isinstance(data, dict)
                       and set(data) == {"ref", "sha"}
                       and isinstance(data.get("ref"), str) and isinstance(data.get("sha"), str)
                       and REF_PATTERN.fullmatch(data.get("ref", "")) is not None
                       and SHA_PATTERN.fullmatch(data.get("sha", "")) is not None))
        if not allowed or phase not in PHASES:
            raise ProofError("Refusing an endpoint outside the fixed proof protocol")
        url = API + path
        request = urllib.request.Request(url, method=method, data=None if data is None else json.dumps(data).encode(), headers={
            "Authorization": "Bearer " + self._token,
            "Accept": "application/vnd.github+json",
            "Content-Type": "application/json",
            "X-GitHub-Api-Version": API_VERSION,
            "User-Agent": "gz-quality-tap-write-proof",
        })
        try:
            try:
                response = self._opener.open(request, timeout=min(REQUEST_TIMEOUT, self._remaining(cleanup)))
            except urllib.error.HTTPError as error:
                response = error
            with response:
                if response.geturl() != url:
                    raise ProofError("Refusing a redirected GitHub API response")
                status = response.code
                if not isinstance(status, int) or isinstance(status, bool) or not 100 <= status <= 599:
                    raise ProofError("GitHub API returned an invalid HTTP status")
                print(f"Tap proof API phase={phase} method={method} path={path} HTTP={status}", flush=True)
                body = bytearray()
                while True:
                    self._remaining(cleanup)
                    chunk = response.read1(min(4096, MAX_BODY + 1 - len(body)))
                    if not chunk:
                        break
                    body.extend(chunk)
                    if len(body) > MAX_BODY:
                        raise ProofError("GitHub API response exceeded the size limit")
                self._remaining(cleanup)
                return status, bytes(body)
        except ProofError:
            print(f"Tap proof API phase={phase} method={method} path={path} failure=protocol-or-deadline", flush=True)
            raise
        except Exception:
            print(f"Tap proof API phase={phase} method={method} path={path} failure=transport HTTP=unavailable", flush=True)
            raise ProofError("GitHub API transport failed") from None


def read_path(reference):
    return "git/ref/" + reference.removeprefix("refs/")


def reconcile(client, reference):
    """Read-only reconciliation never grants ownership after an uncertain POST."""
    try:
        reply = client.request("GET", read_path(reference), cleanup=True, phase="reconcile")
        if absent(reply):
            return "absent"
        value = object_body(reply[1])
        obj = value.get("object")
        if reply[0] == 200 and value.get("ref") == reference and isinstance(obj, dict) and obj.get("type") == "commit" and isinstance(obj.get("sha"), str) and SHA_PATTERN.fullmatch(obj["sha"]):
            return "present"
        return "uncertain"
    except Exception:
        return "uncertain"


def cleanup_owned(client, reference, sha):
    # A matching current ref is mandatory immediately before the sole DELETE.
    if not matching(client.request("GET", read_path(reference), cleanup=True, phase="cleanup-check"), reference, sha):
        raise ProofError("Cleanup refused: current ref ownership is not proven")
    try:
        deleted = client.request("DELETE", "git/refs/" + reference.removeprefix("refs/"), cleanup=True, phase="cleanup-delete")
    except Exception:
        state = reconcile(client, reference)
        raise ProofError("Cleanup DELETE was not acknowledged; ref state=" + state) from None
    if deleted != (204, b""):
        state = reconcile(client, reference)
        raise ProofError("Cleanup DELETE response was invalid; ref state=" + state)
    if not absent(client.request("GET", read_path(reference), cleanup=True, phase="cleanup-absence")):
        raise ProofError("Cleanup absence was not verified")


def verify(client, reference):
    if REF_PATTERN.fullmatch(reference) is None:
        raise ProofError("Invalid temporary reference identity")
    main = client.request("GET", "git/ref/heads/main", phase="main")
    try:
        value = object_body(main[1])
        sha = value["object"]["sha"]
        if not isinstance(sha, str) or SHA_PATTERN.fullmatch(sha) is None or not matching(main, "refs/heads/main", sha):
            raise ValueError("invalid main")
    except Exception:
        raise ProofError("Tap main commit could not be verified") from None
    if not absent(client.request("GET", read_path(reference), phase="absence")):
        raise ProofError("Temporary ref absence not proven; no mutation performed; ref=" + reference)
    # The unique ref is logged before the only POST, so interrupted/uncertain
    # operations remain discoverable without exposing a response or credential.
    print("Tap proof temporary ref: " + reference, flush=True)
    try:
        created = client.request("POST", "git/refs", {"ref": reference, "sha": sha}, phase="create")
        owned = matching(created, reference, sha, status=201)
    except Exception:
        owned = False
    if not owned:
        state = reconcile(client, reference)
        raise ProofError("Creation not proven; no DELETE attempted; ref=" + reference + "; state=" + state)
    verification_failed = False
    try:
        observed = client.request("GET", read_path(reference), phase="verify")
    except Exception:
        verification_failed = True
    else:
        if observed[0] != 200:
            verification_failed = True
        elif not matching(observed, reference, sha):
            # A mismatched observed ref may belong to another writer: never delete.
            raise ProofError("Created ref changed or is malformed; no DELETE attempted; ref=" + reference)
    try:
        cleanup_owned(client, reference, sha)
    except Exception:
        raise ProofError("Cleanup failed or ownership became uncertain; stop publication; inspect ref=" + reference) from None
    if verification_failed:
        raise ProofError("Verification failed; owned ref safely cleaned; stop publication")
    return sha


def main():
    try:
        if os.environ.get("GITHUB_ACTIONS") != "true" or os.environ.get("GITHUB_REPOSITORY") != SOURCE:
            raise ProofError("This proof may run only in the fixed source repository's GitHub Actions")
        run = os.environ.get("GITHUB_RUN_ID", "")
        attempt = os.environ.get("GITHUB_RUN_ATTEMPT", "")
        if re.fullmatch(r"[1-9][0-9]{0,19}", run) is None or re.fullmatch(r"[1-9][0-9]{0,5}", attempt) is None:
            raise ProofError("GitHub Actions run identity is missing or invalid")
        token = os.environ.get("HOMEBREW_TAP_TOKEN", "")
        if not token or any(c.isspace() for c in token):
            raise ProofError("HOMEBREW_TAP_TOKEN is missing or malformed")
        reference = PREFIX + run + "-" + attempt + "-" + secrets.token_hex(16)
        sha = verify(Client(token), reference)
        print("Tap credential write proof PASS: " + REPOSITORY + " main=" + sha + "; temporary ref created, verified and removed")
        return 0
    except ProofError as error:
        print("Tap credential proof failed: " + str(error), file=sys.stderr)
        return 1
    except Exception:
        print("Tap credential proof failed: internal failure (details redacted)", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
