"""Full billing cycle: create → draft → finalize → cancel (storno) → delete draft."""
import pytest
from helpers.api import APIError


@pytest.fixture(scope="module")
def billing_setup(api, test_eeg, test_members, test_meter_points, test_readings):
    """Ensure EEG + members + meter points + readings exist."""
    return test_eeg["id"]


class TestBillingCycle:
    def test_create_billing_run(self, api, billing_setup):
        eeg_id = billing_setup
        run = api.create_billing_run(eeg_id, {
            "period_start": "2025-01-01",
            "period_end": "2025-01-31",
        })
        assert run["status"] == "draft"
        assert run["id"]
        self.__class__._run_id = run["id"]

    def test_billing_run_has_invoices(self, api, billing_setup):
        eeg_id = billing_setup
        invoices = api.list_invoices(eeg_id, self.__class__._run_id)
        assert len(invoices) >= 1, "Expected at least one invoice"
        for inv in invoices:
            assert inv["total_amount"] is not None

    def test_duplicate_billing_run_rejected(self, api, billing_setup):
        """Overlapping billing run must return 409."""
        eeg_id = billing_setup
        with pytest.raises(APIError) as exc:
            api.create_billing_run(eeg_id, {
                "period_start": "2025-01-15",
                "period_end": "2025-01-20",
            })
        assert exc.value.status_code == 409

    def test_finalize_billing_run(self, api, billing_setup):
        eeg_id = billing_setup
        run = api.finalize_billing_run(eeg_id, self.__class__._run_id)
        assert run["status"] == "finalized"

    def test_cannot_delete_finalized_run(self, api, billing_setup):
        """DELETE on a finalized run must be rejected (only draft can be deleted)."""
        eeg_id = billing_setup
        with pytest.raises(APIError) as exc:
            api.delete_billing_run(eeg_id, self.__class__._run_id)
        assert exc.value.status_code in (400, 409, 422)

    def test_cancel_billing_run_creates_storno(self, api, billing_setup):
        """Cancelling a finalized run transitions it to 'cancelled'."""
        eeg_id = billing_setup
        run = api.cancel_billing_run(eeg_id, self.__class__._run_id)
        assert run["status"] == "cancelled"

    def test_invoices_have_storno_pdf_after_cancel(self, api, billing_setup):
        eeg_id = billing_setup
        invoices = api.list_invoices(eeg_id, self.__class__._run_id)
        for inv in invoices:
            assert inv.get("storno_pdf_path"), (
                f"Invoice {inv['id']} has no storno_pdf_path after cancel"
            )


class TestDraftDelete:
    """A draft run can be hard-deleted without going through finalize."""

    def test_create_and_delete_draft(self, api, test_eeg, test_readings):
        eeg_id = test_eeg["id"]
        run = api.create_billing_run(eeg_id, {
            "period_start": "2025-02-01",
            "period_end": "2025-02-28",
        })
        assert run["status"] == "draft"
        run_id = run["id"]

        api.delete_billing_run(eeg_id, run_id)

        # Run must no longer appear in list.
        runs = api.list_billing_runs(eeg_id)
        ids = [r["id"] for r in runs]
        assert run_id not in ids


class TestInvoicePDF:
    def test_invoice_pdf_is_pdf(self, api, test_eeg, test_readings):
        """Create a finalized run and verify the PDF bytes start with %PDF."""
        eeg_id = test_eeg["id"]
        run = api.create_billing_run(eeg_id, {
            "period_start": "2025-03-01",
            "period_end": "2025-03-31",
        })
        api.finalize_billing_run(eeg_id, run["id"])
        invoices = api.list_invoices(eeg_id, run["id"])
        assert invoices, "No invoices generated"

        pdf = api.get_invoice_pdf(eeg_id, invoices[0]["id"])
        assert pdf[:4] == b"%PDF", "Response is not a PDF"

        # Cleanup: cancel the run.
        api.cancel_billing_run(eeg_id, run["id"])
