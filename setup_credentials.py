"""
Pre-creates test credentials in SAP Credential Store for ESO E2E testing.
Uses credstore.py CredentialStore client.

Supported types on this SAP CS instance: password, key, keyring
(certificate type is not supported by this API version)

Requires credstore-sk.json in the current directory.

Note: key values must be base64-encoded strings.
"""
import base64
import subprocess
from credstore import CredentialStore

with CredentialStore("credstore-sk.json") as cs:
    NS = "eso-test"

    print("=== Creating test credentials ===")

    # Password
    try:
        cs.create_password(NS, "db-password", "s3cr3t-pw", username="dbadmin")
        print("  POST password/db-password                → OK")
    except RuntimeError as e:
        print(f"  POST password/db-password                → FAILED: {e}")

    # Key — value must be base64-encoded
    try:
        val_b64 = base64.b64encode(b"my-token-abc123").decode()
        cs.create_key(NS, "api-key", val_b64)
        print("  POST key/api-key                         → OK")
        print(f"         (value base64-encoded: {val_b64})")
    except RuntimeError as e:
        print(f"  POST key/api-key                         → FAILED: {e}")

    print("\n=== Verifying credentials exist ===")
    passwords = cs.list_passwords(NS)
    print(f"  password: {[p['name'] for p in passwords]}")

    keys = cs.list_keys(NS)
    print(f"  key:      {[k['name'] for k in keys]}")

    print("\n=== Reading back values ===")
    pw = cs.read_password(NS, "db-password")
    print(f"  db-password.value    = {pw.get('value')!r}")
    print(f"  db-password.username = {pw.get('username')!r}")

    key = cs.read_key(NS, "api-key")
    raw_val = key.get("value", "")
    try:
        decoded = base64.b64decode(raw_val).decode()
    except Exception:
        decoded = raw_val
    print(f"  api-key.value (decoded) = {decoded!r}")
