"""Shared pytest fixtures for the eegabrechnung integration test suite.

Environment variables (with defaults for local docker compose stack):
  API_URL        - Go API base URL             (default: http://localhost:8101)
  WEB_URL        - Next.js web base URL        (default: http://localhost:3001)
  WORKER_URL     - EDA worker HTTP URL         (default: http://localhost:8082)
  MAILPIT_URL    - Mailpit HTTP API URL        (default: http://localhost:8025)
  MAILPIT_SMTP   - Mailpit SMTP host:port      (default: localhost:1025)
  ADMIN_EMAIL    - API admin login             (default: admin@eeg.at)
  ADMIN_PASSWORD - API admin password          (default: admin)
  DB_DSN         - Postgres DSN for direct insertion
                   (default: postgresql://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung)
"""
import os
import uuid

import pytest
import psycopg2
import psycopg2.extras

from helpers.api import APIClient
from helpers.mailpit import MailpitClient

API_URL = os.environ.get("API_URL", "http://localhost:8101")
WEB_URL = os.environ.get("WEB_URL", "http://localhost:3001")
WORKER_URL = os.environ.get("WORKER_URL", "http://localhost:8082")
MAILPIT_URL = os.environ.get("MAILPIT_URL", "http://localhost:8025")
MAILPIT_SMTP_HOST = os.environ.get("MAILPIT_SMTP_HOST", "localhost")
MAILPIT_SMTP_PORT = int(os.environ.get("MAILPIT_SMTP_PORT", "1025"))
ADMIN_EMAIL = os.environ.get("ADMIN_EMAIL", "admin@eeg.at")
ADMIN_PASSWORD = os.environ.get("ADMIN_PASSWORD", "admin")
DB_DSN = os.environ.get(
    "DB_DSN",
    "postgresql://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung",
)


@pytest.fixture(scope="session")
def api() -> APIClient:
    """Authenticated API client (session-scoped, one login for all tests)."""
    return APIClient.login(API_URL, ADMIN_EMAIL, ADMIN_PASSWORD)


@pytest.fixture(scope="session")
def api_url() -> str:
    return API_URL


@pytest.fixture(scope="session")
def worker_url() -> str:
    return WORKER_URL


@pytest.fixture(scope="session")
def mailpit() -> MailpitClient:
    return MailpitClient(MAILPIT_URL)


@pytest.fixture(scope="session")
def mailpit_smtp() -> tuple[str, int]:
    return (MAILPIT_SMTP_HOST, MAILPIT_SMTP_PORT)


# ── Test EEG + data ──────────────────────────────────────────────────────────

# Unique suffix so parallel test runs don't collide.
_RUN_ID = str(uuid.uuid4())[:8]

# Zählpunkt identifiers for the test meter points.
TEST_ZAEHLPUNKT_CONSUMER = f"AT0010000000000000000000000TEST{_RUN_ID[:4]}C"
TEST_ZAEHLPUNKT_PROSUMER = f"AT0010000000000000000000000TEST{_RUN_ID[:4]}P"
TEST_ZAEHLPUNKT_PRODUCER = f"AT0010000000000000000000000TEST{_RUN_ID[:4]}G"

# EDA IDs for the test EEG (use valid-looking AT numbers).
TEST_GEMEINSCHAFT_ID = f"AT0010000000000000000000000TEST{_RUN_ID[:4]}E"
TEST_MARKTPARTNER_ID = "AT001000000000000000000000000000001"
TEST_NETZBETREIBER_ID = "AT001000000000000000000000000000002"

# Email addresses used in MAIL transport tests.
WORKER_EMAIL = "eda-worker@test.eeg.at"
NETZBETREIBER_EMAIL = "netzbetreiber@test.at"


@pytest.fixture(scope="session")
def test_eeg(api: APIClient) -> dict:
    """Create a test EEG with EDA settings; delete it after the session."""
    suffix = _RUN_ID
    # EDA fields (eda_marktpartner_id, eda_netzbetreiber_id) are only persisted on
    # Update, not Create (the Create INSERT doesn't include them). So we create
    # with basic fields and then update to add EDA configuration.
    eeg = api.create_eeg({
        "name": f"[AUTOTEST] Test-EEG {suffix}",
        "netzbetreiber": "Testnetz GmbH",
        "gemeinschaft_id": TEST_GEMEINSCHAFT_ID,
        "energy_price": 0.12,
        "producer_price": 0.08,
        "use_vat": True,
        "vat_pct": 20.0,
        "billing_period": "monthly",
        "iban": "AT611904300234573201",
        "bic": "RLNWATWWWAI",
        "sepa_creditor_id": "AT12ZZZ00000000001",
    })
    # Set EDA fields via update.
    eeg = api.update_eeg(eeg["id"], {
        **eeg,
        "eda_marktpartner_id": TEST_MARKTPARTNER_ID,
        "eda_netzbetreiber_id": TEST_NETZBETREIBER_ID,
    })
    yield eeg
    # Best-effort teardown — delete billing runs, invoices, members, EEG.
    try:
        _teardown_eeg(api, eeg["id"])
    except Exception as e:
        print(f"[conftest] teardown warning: {e}")


def _teardown_eeg(api: APIClient, eeg_id: str) -> None:
    """Cancel/delete billing runs so they don't interfere with future test runs.

    Safety guard: only operates on EEGs whose name starts with '[AUTOTEST]'.
    Note: the EEG itself is not deleted (no DELETE /eegs endpoint). The test EEG
    will remain in the DB but is uniquely named with the run ID so it won't conflict.
    """
    try:
        eeg = api.get_eeg(eeg_id)
        if not eeg.get("name", "").startswith("[AUTOTEST]"):
            print(f"[conftest] SAFETY: skipping teardown for non-test EEG {eeg_id}")
            return
    except Exception:
        return
    try:
        runs = api.list_billing_runs(eeg_id)
    except Exception:
        runs = []
    for run in runs:
        try:
            status = run["status"]
            if status == "draft":
                api.delete_billing_run(eeg_id, run["id"])
            elif status == "finalized":
                api.cancel_billing_run(eeg_id, run["id"])
            # cancelled runs are already terminal — leave them
        except Exception:
            pass


@pytest.fixture(scope="session")
def test_members(api: APIClient, test_eeg: dict) -> dict:
    """Create three test members (consumer, prosumer, producer)."""
    eeg_id = test_eeg["id"]
    consumer = api.create_member(eeg_id, {
        "name1": "Test Verbraucher",
        "email": "consumer@test.at",
        "status": "ACTIVE",
    })
    prosumer = api.create_member(eeg_id, {
        "name1": "Test Prosumer",
        "email": "prosumer@test.at",
        "status": "ACTIVE",
    })
    producer = api.create_member(eeg_id, {
        "name1": "Test Erzeuger GmbH",
        "email": "producer@test.at",
        "status": "ACTIVE",
    })
    return {"consumer": consumer, "prosumer": prosumer, "producer": producer}


@pytest.fixture(scope="session")
def test_meter_points(api: APIClient, test_eeg: dict, test_members: dict) -> dict:
    """Create one meter point per member."""
    eeg_id = test_eeg["id"]
    mp_consumer = api.create_meter_point(eeg_id, test_members["consumer"]["id"], {
        "zaehlpunkt": TEST_ZAEHLPUNKT_CONSUMER,
        "energierichtung": "CONSUMPTION",
    })
    mp_prosumer = api.create_meter_point(eeg_id, test_members["prosumer"]["id"], {
        "zaehlpunkt": TEST_ZAEHLPUNKT_PROSUMER,
        "energierichtung": "CONSUMPTION",
    })
    mp_producer = api.create_meter_point(eeg_id, test_members["producer"]["id"], {
        "zaehlpunkt": TEST_ZAEHLPUNKT_PRODUCER,
        "energierichtung": "GENERATION",
    })
    return {
        "consumer": mp_consumer,
        "prosumer": mp_prosumer,
        "producer": mp_producer,
    }


@pytest.fixture(scope="session")
def db_conn():
    """Raw psycopg2 connection for direct DB queries (test-internal use only)."""
    conn = psycopg2.connect(DB_DSN)
    conn.autocommit = True
    yield conn
    conn.close()


@pytest.fixture(scope="session")
def test_readings(test_eeg: dict, test_meter_points: dict) -> None:
    """Insert energy readings for 2025-01..05 directly into the DB.

    Uses psycopg2 for direct insertion since there is no bulk readings API endpoint.
    The readings cover Jan–May 2025 so multiple billing periods can be tested.
    """
    mp_consumer = test_meter_points["consumer"]["id"]
    mp_prosumer = test_meter_points["prosumer"]["id"]
    mp_producer = test_meter_points["producer"]["id"]

    rows = []
    for month in range(1, 6):  # Jan–May 2025
        days_in_month = [31, 28, 31, 30, 31][month - 1]
        for day in range(1, days_in_month + 1):
            for hour in range(24):
                ts = f"2025-{month:02d}-{day:02d}T{hour:02d}:00:00+00:00"
                rows.append((mp_consumer, ts, 500, 300, 200, "test"))
                rows.append((mp_prosumer, ts, 600, 400, 200, "test"))
                rows.append((mp_producer, ts, 1200, 800, 0, "test"))

    conn = psycopg2.connect(DB_DSN)
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
    finally:
        conn.close()
