"""Typed API client for the eegabrechnung Go API (direct port 8101)."""
import io

import requests


class APIError(Exception):
    def __init__(self, status_code: int, body: dict):
        self.status_code = status_code
        self.body = body
        super().__init__(f"HTTP {status_code}: {body}")


class APIClient:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.session = requests.Session()
        self.session.headers["Authorization"] = f"Bearer {token}"

    def _url(self, path: str) -> str:
        return f"{self.base_url}{path}"

    def _check(self, resp: requests.Response) -> dict:
        if resp.status_code >= 400:
            try:
                body = resp.json()
            except Exception:
                body = {"raw": resp.text}
            raise APIError(resp.status_code, body)
        if resp.status_code == 204:
            return {}
        return resp.json()

    # ── Auth ─────────────────────────────────────────────────────────────────

    @classmethod
    def login(cls, base_url: str, email: str, password: str) -> "APIClient":
        resp = requests.post(
            f"{base_url.rstrip('/')}/api/v1/auth/login",
            json={"email": email, "password": password},
        )
        resp.raise_for_status()
        token = resp.json()["token"]
        return cls(base_url, token)

    # ── EEGs ─────────────────────────────────────────────────────────────────

    def create_eeg(self, data: dict) -> dict:
        return self._check(self.session.post(self._url("/api/v1/eegs"), json=data))

    def get_eeg(self, eeg_id: str) -> dict:
        return self._check(self.session.get(self._url(f"/api/v1/eegs/{eeg_id}")))

    def update_eeg(self, eeg_id: str, data: dict) -> dict:
        return self._check(self.session.put(self._url(f"/api/v1/eegs/{eeg_id}"), json=data))

    # ── Members ───────────────────────────────────────────────────────────────

    def create_member(self, eeg_id: str, data: dict) -> dict:
        return self._check(
            self.session.post(self._url(f"/api/v1/eegs/{eeg_id}/members"), json=data)
        )

    def list_members(self, eeg_id: str) -> list:
        return self._check(self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/members")))

    def delete_member(self, eeg_id: str, member_id: str) -> None:
        self._check(
            self.session.delete(self._url(f"/api/v1/eegs/{eeg_id}/members/{member_id}"))
        )

    # ── Meter points ─────────────────────────────────────────────────────────

    def create_meter_point(self, eeg_id: str, member_id: str, data: dict) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/members/{member_id}/meter-points"),
                json=data,
            )
        )

    # ── Energy readings ───────────────────────────────────────────────────────

    def bulk_insert_readings(self, eeg_id: str, readings: list[dict]) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/readings/bulk"),
                json=readings,
            )
        )

    # ── Billing runs ──────────────────────────────────────────────────────────

    def create_billing_run(self, eeg_id: str, data: dict) -> dict:
        """Create a billing run. Returns the billing_run dict (not the full wrapper)."""
        resp = self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/billing/run"), json=data
            )
        )
        # Response: {"billing_run": {...}, "invoices": [...], "invoices_created": N}
        return resp["billing_run"]

    def list_billing_runs(self, eeg_id: str) -> list:
        return self._check(
            self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/billing/runs"))
        )

    def get_billing_run(self, eeg_id: str, run_id: str) -> dict:
        return self._check(
            self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/billing/runs/{run_id}"))
        )

    def finalize_billing_run(self, eeg_id: str, run_id: str) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/billing/runs/{run_id}/finalize")
            )
        )

    def cancel_billing_run(self, eeg_id: str, run_id: str) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/billing/runs/{run_id}/cancel")
            )
        )

    def delete_billing_run(self, eeg_id: str, run_id: str) -> None:
        self._check(
            self.session.delete(
                self._url(f"/api/v1/eegs/{eeg_id}/billing/runs/{run_id}")
            )
        )

    # ── Invoices ──────────────────────────────────────────────────────────────

    def list_invoices(self, eeg_id: str, run_id: str) -> list:
        return self._check(
            self.session.get(
                self._url(f"/api/v1/eegs/{eeg_id}/billing/runs/{run_id}/invoices")
            )
        )

    def get_invoice_pdf(self, eeg_id: str, invoice_id: str) -> bytes:
        resp = self.session.get(
            self._url(f"/api/v1/eegs/{eeg_id}/invoices/{invoice_id}/pdf")
        )
        if resp.status_code >= 400:
            raise APIError(resp.status_code, {"raw": resp.text})
        return resp.content

    # ── SEPA ─────────────────────────────────────────────────────────────────

    def sepa_pain001(self, eeg_id: str) -> bytes:
        resp = self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/sepa/pain001"))
        if resp.status_code >= 400:
            raise APIError(resp.status_code, {"raw": resp.text})
        return resp.content

    def sepa_pain008(self, eeg_id: str) -> bytes:
        resp = self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/sepa/pain008"))
        if resp.status_code >= 400:
            raise APIError(resp.status_code, {"raw": resp.text})
        return resp.content

    # ── EDA processes ─────────────────────────────────────────────────────────

    def eda_anmeldung(self, eeg_id: str, data: dict) -> dict:
        resp = self.session.post(
            self._url(f"/api/v1/eegs/{eeg_id}/eda/anmeldung"), json=data
        )
        if resp.status_code not in (200, 201):
            raise APIError(resp.status_code, resp.json())
        return resp.json()

    def eda_abmeldung(self, eeg_id: str, data: dict) -> dict:
        resp = self.session.post(
            self._url(f"/api/v1/eegs/{eeg_id}/eda/abmeldung"), json=data
        )
        if resp.status_code not in (200, 201):
            raise APIError(resp.status_code, resp.json())
        return resp.json()

    def list_eda_processes(self, eeg_id: str) -> list:
        return self._check(
            self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/eda/processes"))
        )

    def get_eda_process(self, eeg_id: str, process_id: str) -> dict:
        procs = self.list_eda_processes(eeg_id)
        for p in procs:
            if p["id"] == process_id:
                return p
        raise KeyError(f"EDA process {process_id} not found")

    # ── Onboarding (public — no auth) ─────────────────────────────────────────

    def get_public_eeg_info(self, eeg_id: str) -> dict:
        return self._check(requests.get(self._url(f"/api/v1/public/eegs/{eeg_id}/info")))

    def submit_onboarding(self, eeg_id: str, data: dict) -> dict:
        resp = requests.post(
            self._url(f"/api/v1/public/eegs/{eeg_id}/onboarding"), json=data
        )
        if resp.status_code not in (200, 201):
            raise APIError(resp.status_code, resp.json())
        return resp.json()

    def get_onboarding_status(self, token: str) -> dict:
        return self._check(
            requests.get(self._url(f"/api/v1/public/onboarding/status/{token}"))
        )

    # ── Onboarding (admin — auth required) ────────────────────────────────────

    def list_onboarding(self, eeg_id: str) -> list:
        return self._check(
            self.session.get(self._url(f"/api/v1/eegs/{eeg_id}/onboarding"))
        )

    def update_onboarding_status(
        self, eeg_id: str, request_id: str, status: str, notes: str = ""
    ) -> dict:
        return self._check(
            self.session.patch(
                self._url(f"/api/v1/eegs/{eeg_id}/onboarding/{request_id}"),
                json={"status": status, "admin_notes": notes},
            )
        )

    # ── Import ────────────────────────────────────────────────────────────────

    def import_stammdaten(
        self, eeg_id: str, xlsx_bytes: bytes, filename: str = "stammdaten.xlsx"
    ) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/import/stammdaten"),
                files={"file": (filename, io.BytesIO(xlsx_bytes), "application/octet-stream")},
            )
        )

    def preview_energiedaten(
        self, eeg_id: str, xlsx_bytes: bytes, filename: str = "energiedaten.xlsx"
    ) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/import/energiedaten/preview"),
                files={"file": (filename, io.BytesIO(xlsx_bytes), "application/octet-stream")},
            )
        )

    def import_energiedaten(
        self,
        eeg_id: str,
        xlsx_bytes: bytes,
        mode: str = "overwrite",
        filename: str = "energiedaten.xlsx",
    ) -> dict:
        return self._check(
            self.session.post(
                self._url(f"/api/v1/eegs/{eeg_id}/import/energiedaten?mode={mode}"),
                files={"file": (filename, io.BytesIO(xlsx_bytes), "application/octet-stream")},
            )
        )

    def get_coverage(self, eeg_id: str, year: int) -> dict:
        return self._check(
            self.session.get(
                self._url(f"/api/v1/eegs/{eeg_id}/readings/coverage?year={year}")
            )
        )

    # ── Accounting export ─────────────────────────────────────────────────────

    def accounting_export(
        self, eeg_id: str, from_date: str, to_date: str, fmt: str = "xlsx"
    ) -> bytes:
        resp = self.session.get(
            self._url(
                f"/api/v1/eegs/{eeg_id}/accounting/export"
                f"?from={from_date}&to={to_date}&format={fmt}"
            )
        )
        if resp.status_code >= 400:
            raise APIError(resp.status_code, {"raw": resp.text})
        return resp.content
