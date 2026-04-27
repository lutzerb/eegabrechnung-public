"""SEPA pain.001 and pain.008 XML validation.

SEPA files are generated per-EEG (all finalized, unpaid invoices).
"""
import pytest
from lxml import etree

from helpers.api import APIError


@pytest.fixture(scope="module")
def eeg_with_invoices(api, test_eeg, test_readings):
    """Ensure test EEG has at least one finalized billing run with invoices."""
    eeg_id = test_eeg["id"]
    # Create and finalize a billing run so there are invoices in DB.
    try:
        run = api.create_billing_run(eeg_id, {
            "period_start": "2025-04-01",
            "period_end": "2025-04-30",
        })
        api.finalize_billing_run(eeg_id, run["id"])
    except APIError as e:
        if e.status_code == 409:
            pass  # Already exists from another test
        else:
            raise
    yield eeg_id
    # Cancel if still finalized.
    try:
        runs = api.list_billing_runs(eeg_id)
        for r in runs:
            if r["status"] == "finalized":
                try:
                    api.cancel_billing_run(eeg_id, r["id"])
                except Exception:
                    pass
    except Exception:
        pass


class TestSEPA:
    def test_pain001_is_valid_xml(self, api, eeg_with_invoices):
        eeg_id = eeg_with_invoices
        try:
            xml_bytes = api.sepa_pain001(eeg_id)
        except APIError as e:
            if e.status_code in (404, 422):
                pytest.skip("No SEPA credit transactions (members have no IBAN)")
            raise
        root = etree.fromstring(xml_bytes)
        assert root is not None

    def test_pain008_is_valid_xml(self, api, eeg_with_invoices):
        eeg_id = eeg_with_invoices
        try:
            xml_bytes = api.sepa_pain008(eeg_id)
        except APIError as e:
            if e.status_code in (404, 422):
                pytest.skip("No SEPA debit transactions (members have no IBAN)")
            raise
        root = etree.fromstring(xml_bytes)
        assert root is not None

    def test_pain001_contains_creditor_id(self, api, eeg_with_invoices, test_eeg):
        eeg_id = eeg_with_invoices
        try:
            xml_bytes = api.sepa_pain001(eeg_id)
        except APIError as e:
            if e.status_code in (404, 422):
                pytest.skip("No pain.001 transactions")
            raise
        xml_str = xml_bytes.decode("utf-8")
        eeg = api.get_eeg(eeg_id)
        if eeg.get("sepa_creditor_id"):
            assert eeg["sepa_creditor_id"] in xml_str
