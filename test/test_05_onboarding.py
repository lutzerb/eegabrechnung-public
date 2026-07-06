"""Onboarding flow: public application → admin approval → member created.

Tests both the public (unauthenticated) onboarding endpoints and the admin
approval flow that creates a member + meter points (+ optional EDA process).
"""
import uuid as _uuid

import pytest
import psycopg2

from conftest import DB_DSN
from helpers.api import APIClient, APIError


def _get_token_from_db(request_id: str) -> str:
    """Fetch the magic_token directly from the DB (it's not exposed via API)."""
    conn = psycopg2.connect(DB_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT magic_token FROM onboarding_requests WHERE id = %s",
                (request_id,),
            )
            row = cur.fetchone()
            return row[0] if row else ""
    finally:
        conn.close()


class TestPublicEEGInfo:
    """GET /api/v1/public/eegs/{eegID}/info — no auth required."""

    def test_returns_name_and_billing_period(self, api, test_eeg):
        info = api.get_public_eeg_info(test_eeg["id"])
        assert info["id"] == test_eeg["id"]
        assert info["name"] == test_eeg["name"]
        assert "billing_period" in info

    def test_unknown_eeg_returns_404(self, api):
        import requests as _req
        resp = _req.get(
            f"http://localhost:8101/api/v1/public/eegs/{_uuid.uuid4()}/info"
        )
        assert resp.status_code == 404


class TestOnboardingSubmit:
    """POST /api/v1/public/eegs/{eegID}/onboarding — validation + happy path."""

    def test_missing_name_returns_400(self, api, test_eeg):
        import requests as _req
        resp = _req.post(
            f"http://localhost:8101/api/v1/public/eegs/{test_eeg['id']}/onboarding",
            json={"email": "x@test.at", "contract_accepted": True},
        )
        assert resp.status_code == 400

    def test_missing_email_returns_400(self, api, test_eeg):
        import requests as _req
        resp = _req.post(
            f"http://localhost:8101/api/v1/public/eegs/{test_eeg['id']}/onboarding",
            json={"name1": "Max", "contract_accepted": True},
        )
        assert resp.status_code == 400

    def test_contract_not_accepted_returns_400(self, api, test_eeg):
        import requests as _req
        resp = _req.post(
            f"http://localhost:8101/api/v1/public/eegs/{test_eeg['id']}/onboarding",
            json={"name1": "Max", "email": "max@test.at", "contract_accepted": False},
        )
        assert resp.status_code == 400

    def test_valid_submission_returns_201(self, api, test_eeg):
        uid = _uuid.uuid4().hex[:6]
        result = api.submit_onboarding(test_eeg["id"], {
            "name1": f"Onboarding Test {uid}",
            "email": f"ob_{uid}@test.at",
            "contract_accepted": True,
            "member_type": "CONSUMER",
        })
        assert result["status"] == "pending"
        assert "id" in result
        self.__class__._request_id = result["id"]

    def test_status_endpoint_requires_valid_token(self, api):
        import requests as _req
        resp = _req.get(
            "http://localhost:8101/api/v1/public/onboarding/status/not-a-real-token"
        )
        assert resp.status_code in (404, 410)


class TestOnboardingAdminFlow:
    """Full approval cycle: submit → admin list → reject / approve → member created."""

    @pytest.fixture(autouse=True)
    def submit_request(self, api, test_eeg):
        uid = _uuid.uuid4().hex[:6]
        zp = f"AT001000000000000000000000OBTEST{uid[:4]}C"
        result = api.submit_onboarding(test_eeg["id"], {
            "name1": f"Ob Approve {uid}",
            "email": f"ob_approve_{uid}@test.at",
            "contract_accepted": True,
            "beitritts_datum": "2026-01-01",
            "meter_points": [{"zaehlpunkt": zp, "direction": "CONSUMPTION"}],
        })
        self.request_id = result["id"]
        self.eeg_id = test_eeg["id"]
        self.zaehlpunkt = zp

    def test_admin_can_list_pending_requests(self, api):
        reqs = api.list_onboarding(self.eeg_id)
        ids = [r["id"] for r in reqs]
        assert self.request_id in ids

    def test_status_visible_via_token(self, api):
        token = _get_token_from_db(self.request_id)
        assert token, "token not found in DB"
        status = api.get_onboarding_status(token)
        assert status["status"] == "pending"
        assert status["name1"].startswith("Ob Approve")

    def test_reject_sets_status_rejected(self, api):
        result = api.update_onboarding_status(
            self.eeg_id, self.request_id, "rejected", notes="Test rejection"
        )
        assert result["status"] == "rejected"

    def test_approve_creates_member_and_sets_converted(self, api, test_eeg):
        uid = _uuid.uuid4().hex[:6]
        zp = f"AT001000000000000000000000OBAPPRV{uid[:4]}C"
        result = api.submit_onboarding(test_eeg["id"], {
            "name1": f"Ob Converted {uid}",
            "email": f"ob_conv_{uid}@test.at",
            "contract_accepted": True,
            "beitritts_datum": "2026-02-01",
            "meter_points": [{"zaehlpunkt": zp, "direction": "CONSUMPTION"}],
        })
        req_id = result["id"]

        approved = api.update_onboarding_status(
            test_eeg["id"], req_id, "approved"
        )
        assert approved["status"] == "converted"
        assert approved.get("converted_member_id"), "converted_member_id should be set"

        # Verify the member actually appears in the members list.
        members = api.list_members(test_eeg["id"])
        names = [m["name1"] for m in members]
        assert f"Ob Converted {uid}" in names, (
            f"Expected converted member in list, got: {names}"
        )

    def test_double_approve_returns_409(self, api, test_eeg):
        """Approving an already-converted request must return 409."""
        uid = _uuid.uuid4().hex[:6]
        result = api.submit_onboarding(test_eeg["id"], {
            "name1": f"Ob Double {uid}",
            "email": f"ob_double_{uid}@test.at",
            "contract_accepted": True,
        })
        req_id = result["id"]
        api.update_onboarding_status(test_eeg["id"], req_id, "approved")

        with pytest.raises(APIError) as exc:
            api.update_onboarding_status(test_eeg["id"], req_id, "approved")
        assert exc.value.status_code == 409
