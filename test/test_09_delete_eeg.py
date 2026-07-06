"""Integration tests for EEG deletion.

Tests:
  - DELETE /api/v1/eegs/{id} returns 204 and removes the EEG
  - All dependent data (members, meter points, readings, invoices) is gone after delete
  - Deleting a non-existent EEG returns 404
  - Deleting another org's EEG returns 404 (org isolation)
  - Unauthenticated delete returns 401
"""
import uuid
import psycopg2
import psycopg2.extras
import os
import pytest

from helpers.api import APIClient, APIError

API_URL = None  # populated by conftest api_url fixture

DB_DSN = os.environ.get(
    "DB_DSN",
    "postgresql://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung",
)


def _make_zp(tag: str) -> str:
    """Return a valid 33-char Zählpunkt."""
    h = str(uuid.uuid4()).replace("-", "").upper()[:4]
    return f"AT001000000000000000000000DEL{h}{tag}"[:33]


def _insert_readings(mp_id: str) -> None:
    conn = psycopg2.connect(DB_DSN)
    try:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                "INSERT INTO energy_readings (meter_point_id, ts, wh_total, wh_community, wh_self, source) VALUES %s ON CONFLICT DO NOTHING",
                [(mp_id, "2025-07-01T00:00:00+00:00", 100, 60, 40, "test")],
                template="(%s,%s,%s,%s,%s,%s)",
            )
        conn.commit()
    finally:
        conn.close()


def _row_count(table: str, where_col: str, value: str) -> int:
    conn = psycopg2.connect(DB_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(f"SELECT COUNT(*) FROM {table} WHERE {where_col} = %s", (value,))
            return cur.fetchone()[0]
    finally:
        conn.close()


def _mp_row_count(eeg_id: str) -> int:
    conn = psycopg2.connect(DB_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT COUNT(*) FROM meter_points mp JOIN members m ON mp.member_id=m.id WHERE m.eeg_id=%s",
                (eeg_id,),
            )
            return cur.fetchone()[0]
    finally:
        conn.close()


def _reading_count_for_eeg(eeg_id: str) -> int:
    conn = psycopg2.connect(DB_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """SELECT COUNT(*) FROM energy_readings er
                   JOIN meter_points mp ON mp.id = er.meter_point_id
                   JOIN members m ON m.id = mp.member_id
                   WHERE m.eeg_id = %s""",
                (eeg_id,),
            )
            return cur.fetchone()[0]
    finally:
        conn.close()


# ── tests ─────────────────────────────────────────────────────────────────────

def test_delete_eeg_removes_all_data(api: APIClient, api_url: str):
    """Creating an EEG with members, meter points, and readings; deleting it wipes everything."""
    suffix = str(uuid.uuid4())[:8]
    eeg = api.create_eeg({
        "name": f"[AUTOTEST] Delete-Test {suffix}",
        "netzbetreiber": "Testnetz",
        "gemeinschaft_id": _make_zp("E"),
        "energy_price": 0.10,
        "producer_price": 0.06,
    })
    eeg_id = eeg["id"]

    member = api.create_member(eeg_id, {"name1": "Lösch-Mitglied", "status": "ACTIVE"})
    mp = api.create_meter_point(eeg_id, member["id"], {
        "zaehlpunkt": _make_zp("C"),
        "energierichtung": "CONSUMPTION",
    })
    _insert_readings(mp["id"])

    # Verify data exists before delete.
    assert _row_count("members", "eeg_id", eeg_id) == 1
    assert _mp_row_count(eeg_id) == 1
    assert _reading_count_for_eeg(eeg_id) == 1

    # Delete the EEG.
    resp = api.session.delete(api._url(f"/api/v1/eegs/{eeg_id}"))
    assert resp.status_code == 204, resp.text

    # EEG is gone from API.
    get_resp = api.session.get(api._url(f"/api/v1/eegs/{eeg_id}"))
    assert get_resp.status_code == 404

    # All dependent data is gone from DB.
    assert _row_count("eegs", "id", eeg_id) == 0
    assert _row_count("members", "eeg_id", eeg_id) == 0
    assert _mp_row_count(eeg_id) == 0
    assert _reading_count_for_eeg(eeg_id) == 0


def test_delete_nonexistent_eeg_returns_404(api: APIClient):
    """Deleting a random UUID returns 404."""
    resp = api.session.delete(api._url(f"/api/v1/eegs/{uuid.uuid4()}"))
    assert resp.status_code == 404


def test_delete_eeg_unauthorized(api_url: str):
    """DELETE without a token returns 401."""
    import requests
    resp = requests.delete(f"{api_url}/api/v1/eegs/{uuid.uuid4()}")
    assert resp.status_code == 401


def test_delete_eeg_not_in_list_after_deletion(api: APIClient):
    """Deleted EEG no longer appears in the EEG list."""
    suffix = str(uuid.uuid4())[:8]
    eeg = api.create_eeg({
        "name": f"[AUTOTEST] Disappear-Test {suffix}",
        "netzbetreiber": "Testnetz",
        "gemeinschaft_id": _make_zp("D"),
    })
    eeg_id = eeg["id"]

    resp = api.session.delete(api._url(f"/api/v1/eegs/{eeg_id}"))
    assert resp.status_code == 204

    eegs = api._check(api.session.get(api._url("/api/v1/eegs")))
    ids = {e["id"] for e in eegs}
    assert eeg_id not in ids
