"""EDA process lifecycle tests using FILE transport (no mail server needed).

These tests call the API to create EDA processes and verify the DB state.
They do NOT require the EDA worker to be running — they just check that
the process records are created correctly and can be queried.
"""
import pytest
from helpers.api import APIError
from conftest import TEST_ZAEHLPUNKT_CONSUMER, TEST_ZAEHLPUNKT_PROSUMER


class TestEDAProcessCreation:
    def test_anmeldung_requires_eda_settings(self, api, test_eeg, test_meter_points):
        """Anmeldung succeeds because the test EEG has EDA settings configured."""
        eeg_id = test_eeg["id"]
        proc = api.eda_anmeldung(eeg_id, {
            "zaehlpunkt": TEST_ZAEHLPUNKT_CONSUMER,
            "valid_from": "2025-06-01",
            "share_type": "GC",
            "participation_factor": 50.0,
        })
        assert proc["status"] == "pending"
        assert proc["process_type"] == "EC_EINZEL_ANM"
        assert proc["conversation_id"]
        assert proc["zaehlpunkt"] == TEST_ZAEHLPUNKT_CONSUMER
        self.__class__._anm_proc = proc

    def test_anmeldung_process_appears_in_list(self, api, test_eeg):
        eeg_id = test_eeg["id"]
        procs = api.list_eda_processes(eeg_id)
        ids = [p["id"] for p in procs]
        assert self.__class__._anm_proc["id"] in ids

    def test_anmeldung_without_eda_settings_fails(self, api):
        """A new EEG without eda_marktpartner_id should return 400."""
        import uuid as _uuid
        unique_id = str(_uuid.uuid4())[:8]
        eeg = api.create_eeg({
            "name": f"Test-NoEDA-{unique_id}",
            "netzbetreiber": "Testnetz GmbH",
            "gemeinschaft_id": f"AT001NOEDA{unique_id}",
            "energy_price": 0.1,
            "producer_price": 0.05,
        })
        with pytest.raises(APIError) as exc:
            api.eda_anmeldung(eeg["id"], {
                "zaehlpunkt": f"AT001NOEDA{unique_id}ZPKT",
                "valid_from": "2025-06-01",
            })
        assert exc.value.status_code == 400

    def test_abmeldung_creates_process(self, api, test_eeg, test_meter_points):
        eeg_id = test_eeg["id"]
        proc = api.eda_abmeldung(eeg_id, {
            "zaehlpunkt": TEST_ZAEHLPUNKT_PROSUMER,
            "valid_from": "2025-07-01",
        })
        assert proc["status"] == "pending"
        assert proc["process_type"] == "EC_EINZEL_ABM"
        assert proc["conversation_id"]

    def test_process_has_deadline(self, api, test_eeg):
        """Anmeldung processes must have a deadline (2 months, EAG §16e)."""
        eeg_id = test_eeg["id"]
        procs = api.list_eda_processes(eeg_id)
        anm_procs = [p for p in procs if p["process_type"] == "EC_EINZEL_ANM"]
        assert anm_procs, "No EC_EINZEL_ANM process found"
        for p in anm_procs:
            assert p.get("deadline_at"), f"Process {p['id']} has no deadline_at"
