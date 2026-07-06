"""Basic health and connectivity checks."""
import requests
import pytest

from conftest import API_URL, WORKER_URL


def test_api_health():
    # The health endpoint may or may not require auth depending on config.
    resp = requests.get(f"{API_URL}/api/v1/health", timeout=5)
    assert resp.status_code in (200, 401), f"Unexpected status: {resp.status_code}"
    if resp.status_code == 200:
        assert resp.json().get("status") == "ok"


def test_api_auth_rejects_bad_password():
    resp = requests.post(
        f"{API_URL}/api/v1/auth/login",
        json={"email": "admin@eeg.at", "password": "wrong"},
        timeout=5,
    )
    assert resp.status_code in (401, 403)


def test_api_login_returns_token(api):
    # api fixture already proves login works; just check the token is non-empty.
    assert api.token
    assert len(api.token) > 20


def test_api_list_eegs(api):
    eegs = api._check(api.session.get(f"{API_URL}/api/v1/eegs"))
    assert isinstance(eegs, list)


@pytest.mark.skipif(
    not WORKER_URL,
    reason="WORKER_URL not set — EDA worker test profile not running",
)
def test_worker_health():
    try:
        resp = requests.get(f"{WORKER_URL}/health", timeout=5)
        assert resp.status_code == 200
        assert resp.json().get("status") == "ok"
    except requests.ConnectionError:
        pytest.skip("EDA worker not reachable (start with --profile test)")
