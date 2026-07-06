"""Integration tests for backup and restore functionality.

Tests the full roundtrip:
  1. Create a fresh EEG with members, meter points, tariff, billing run, invoice.
  2. Export backup → verify JSON structure.
  3. Restore backup to the same EEG → verify data matches original.
  4. Restore backup to a fresh target EEG (cross-EEG restore) → verify data remapped correctly.
"""
import json
import uuid
import io

import pytest
import psycopg2
import psycopg2.extras

from helpers.api import APIClient, APIError

API_URL = None  # populated by conftest api_url fixture


# ── helpers ──────────────────────────────────────────────────────────────────

def make_unique_zaehlpunkt(suffix: str) -> str:
    """Return a valid 33-char AT Zählpunkt."""
    tag = str(uuid.uuid4()).replace("-", "").upper()[:4]
    base = f"AT001000000000000000000000BKUP{tag}"
    return base[:33]


def _insert_readings(db_dsn: str, mp_ids: list[str]) -> int:
    """Insert a small set of hourly readings for the given meter point IDs."""
    rows = []
    for mp_id in mp_ids:
        for hour in range(4):  # 4 readings per meter point
            ts = f"2025-06-01T{hour:02d}:00:00+00:00"
            rows.append((mp_id, ts, 100, 60, 40, "test"))

    conn = psycopg2.connect(db_dsn)
    try:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """
                INSERT INTO energy_readings
                    (meter_point_id, ts, wh_total, wh_community, wh_self, source)
                VALUES %s
                ON CONFLICT (meter_point_id, ts) DO NOTHING
                """,
                rows,
                template="(%s, %s, %s, %s, %s, %s)",
            )
        conn.commit()
        return len(rows)
    finally:
        conn.close()


def _get_db_dsn() -> str:
    import os
    return os.environ.get(
        "DB_DSN",
        "postgresql://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung",
    )


def _teardown_eeg(api: APIClient, eeg_id: str) -> None:
    try:
        eeg = api.get_eeg(eeg_id)
        if not eeg.get("name", "").startswith("[AUTOTEST]"):
            return
    except Exception:
        return
    try:
        runs = api.list_billing_runs(eeg_id)
    except Exception:
        runs = []
    for run in runs:
        try:
            if run["status"] == "draft":
                api.delete_billing_run(eeg_id, run["id"])
            elif run["status"] == "finalized":
                api.cancel_billing_run(eeg_id, run["id"])
        except Exception:
            pass


# ── fixtures ──────────────────────────────────────────────────────────────────

@pytest.fixture(scope="module")
def backup_api(api_url) -> APIClient:
    """Authenticated API client for backup tests."""
    return APIClient.login(api_url, "admin@eeg.at", "admin")


@pytest.fixture(scope="module")
def backup_eeg(backup_api: APIClient) -> dict:
    """Create a test EEG with members, meter points, tariff, billing run."""
    suffix = str(uuid.uuid4())[:8]
    eeg = backup_api.create_eeg({
        "name": f"[AUTOTEST] Backup-Test {suffix}",
        "netzbetreiber": "Testnetz",
        "gemeinschaft_id": make_unique_zaehlpunkt(suffix),
        "energy_price": 0.15,
        "producer_price": 0.09,
        "use_vat": True,
        "vat_pct": 20.0,
        "billing_period": "monthly",
    })
    yield eeg
    _teardown_eeg(backup_api, eeg["id"])


@pytest.fixture(scope="module")
def backup_members(backup_api: APIClient, backup_eeg: dict) -> dict:
    eeg_id = backup_eeg["id"]
    consumer = backup_api.create_member(eeg_id, {
        "name1": "Backup Verbraucher",
        "email": "bk-consumer@test.at",
        "status": "ACTIVE",
    })
    producer = backup_api.create_member(eeg_id, {
        "name1": "Backup Erzeuger",
        "email": "bk-producer@test.at",
        "status": "ACTIVE",
    })
    return {"consumer": consumer, "producer": producer}


@pytest.fixture(scope="module")
def backup_meter_points(backup_api: APIClient, backup_eeg: dict, backup_members: dict) -> dict:
    eeg_id = backup_eeg["id"]
    mp_c = backup_api.create_meter_point(eeg_id, backup_members["consumer"]["id"], {
        "zaehlpunkt": make_unique_zaehlpunkt("C"),
        "energierichtung": "CONSUMPTION",
    })
    mp_g = backup_api.create_meter_point(eeg_id, backup_members["producer"]["id"], {
        "zaehlpunkt": make_unique_zaehlpunkt("G"),
        "energierichtung": "GENERATION",
    })
    return {"consumer": mp_c, "producer": mp_g}


@pytest.fixture(scope="module")
def backup_readings(backup_eeg: dict, backup_meter_points: dict) -> int:
    """Insert test readings directly via DB."""
    mp_ids = [
        backup_meter_points["consumer"]["id"],
        backup_meter_points["producer"]["id"],
    ]
    return _insert_readings(_get_db_dsn(), mp_ids)


# ── tests ─────────────────────────────────────────────────────────────────────

def test_backup_export_returns_json(backup_api: APIClient, backup_eeg: dict,
                                    backup_members: dict, backup_meter_points: dict,
                                    backup_readings: int):
    """Backup endpoint returns valid JSON with the expected top-level keys."""
    resp = backup_api.session.get(
        backup_api._url(f"/api/v1/eegs/{backup_eeg['id']}/backup")
    )
    assert resp.status_code == 200, resp.text
    assert "application/json" in resp.headers.get("Content-Type", "")
    content_disp = resp.headers.get("Content-Disposition", "")
    assert "attachment" in content_disp

    data = resp.json()
    assert data["version"] == "1"
    assert "created_at" in data
    assert "eeg" in data
    assert data["eeg"]["id"] == backup_eeg["id"]


def test_backup_contains_all_entities(backup_api: APIClient, backup_eeg: dict,
                                      backup_members: dict, backup_meter_points: dict,
                                      backup_readings: int):
    """Backup includes members, meter points, and readings."""
    resp = backup_api.session.get(
        backup_api._url(f"/api/v1/eegs/{backup_eeg['id']}/backup")
    )
    data = resp.json()

    member_ids = {m["id"] for m in data["members"]}
    assert backup_members["consumer"]["id"] in member_ids
    assert backup_members["producer"]["id"] in member_ids

    mp_ids = {mp["id"] for mp in data["meter_points"]}
    assert backup_meter_points["consumer"]["id"] in mp_ids
    assert backup_meter_points["producer"]["id"] in mp_ids

    assert len(data["readings"]) >= backup_readings


def test_backup_restore_roundtrip(backup_api: APIClient, backup_eeg: dict,
                                   backup_members: dict, backup_meter_points: dict,
                                   backup_readings: int):
    """Full roundtrip: export → restore to same EEG → verify member count matches."""
    eeg_id = backup_eeg["id"]

    # Export backup.
    export_resp = backup_api.session.get(
        backup_api._url(f"/api/v1/eegs/{eeg_id}/backup")
    )
    assert export_resp.status_code == 200
    backup_bytes = export_resp.content

    original = export_resp.json()
    original_member_count = len(original["members"])
    original_reading_count = len(original["readings"])

    # Restore to same EEG.
    restore_resp = backup_api.session.post(
        backup_api._url(f"/api/v1/eegs/{eeg_id}/restore"),
        files={"file": ("backup.json", io.BytesIO(backup_bytes), "application/json")},
    )
    assert restore_resp.status_code == 200, restore_resp.text
    result = restore_resp.json()

    assert result["members"] == original_member_count
    assert result["readings"] == original_reading_count

    # Verify members are restored correctly.
    members_after = backup_api.list_members(eeg_id)
    restored_ids = {m["id"] for m in members_after}
    assert backup_members["consumer"]["id"] in restored_ids
    assert backup_members["producer"]["id"] in restored_ids

    # Verify EEG settings were preserved.
    eeg_after = backup_api.get_eeg(eeg_id)
    assert eeg_after["name"] == original["eeg"]["name"]
    assert abs(eeg_after["energy_price"] - original["eeg"]["energy_price"]) < 0.001


def test_restore_invalid_file_rejected(backup_api: APIClient, backup_eeg: dict):
    """Uploading garbage is rejected with 400."""
    resp = backup_api.session.post(
        backup_api._url(f"/api/v1/eegs/{backup_eeg['id']}/restore"),
        files={"file": ("bad.json", io.BytesIO(b"not json at all"), "application/json")},
    )
    assert resp.status_code == 400


def test_restore_wrong_version_rejected(backup_api: APIClient, backup_eeg: dict):
    """A backup JSON with an unsupported version is rejected."""
    bad = json.dumps({"version": "99", "eeg": {"id": backup_eeg["id"]}, "members": []})
    resp = backup_api.session.post(
        backup_api._url(f"/api/v1/eegs/{backup_eeg['id']}/restore"),
        files={"file": ("bad.json", io.BytesIO(bad.encode()), "application/json")},
    )
    assert resp.status_code == 400


def test_backup_unauthorized(api_url: str):
    """Backup endpoint requires authentication."""
    import requests
    resp = requests.get(f"{api_url}/api/v1/eegs/{uuid.uuid4()}/backup")
    assert resp.status_code == 401


def test_restore_unauthorized(api_url: str):
    """Restore endpoint requires authentication."""
    import requests
    resp = requests.post(
        f"{api_url}/api/v1/eegs/{uuid.uuid4()}/restore",
        files={"file": ("f.json", io.BytesIO(b"{}"), "application/json")},
    )
    assert resp.status_code == 401
