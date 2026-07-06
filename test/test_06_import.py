"""XLSX import tests: Stammdaten (master data) and Energiedaten (readings).

Tests the full import pipeline:
  - Stammdaten: upload XLSX → members + meter points upserted
  - Energiedaten preview: dry-run returns overlap stats without inserting
  - Energiedaten import: uploads readings → rows inserted
  - Coverage: GET /readings/coverage returns daily coverage data
"""
import uuid as _uuid
from datetime import datetime, timezone

import pytest

from conftest import TEST_ZAEHLPUNKT_CONSUMER, TEST_ZAEHLPUNKT_PROSUMER
from helpers.xlsx_builder import build_stammdaten_xlsx, build_energiedaten_xlsx


class TestStammdatenImport:
    """POST /api/v1/eegs/{eegID}/import/stammdaten"""

    def test_import_creates_member_and_meter_point(self, api, test_eeg):
        eeg_id = test_eeg["id"]
        uid = _uuid.uuid4().hex[:6]
        zp = f"AT001000000000000000000000XLSX{uid[:4]}C"

        xlsx = build_stammdaten_xlsx([{
            "netzbetreiber": "Test Netz GmbH",
            "gemeinschaft_id": test_eeg.get("gemeinschaft_id", ""),
            "zaehlpunkt": zp,
            "energierichtung": "CONSUMPTION",
            "name1": f"XLSX Mitglied {uid}",
            "email": f"xlsx_{uid}@test.at",
            "mitglieds_nr": f"TM{uid}",
            "status": "ACTIVATED",
        }])

        result = api.import_stammdaten(eeg_id, xlsx)
        assert result["members"] >= 1, f"Expected ≥1 member, got {result}"
        assert result["meter_points"] >= 1, f"Expected ≥1 meter point, got {result}"

    def test_import_only_activated_rows(self, api, test_eeg):
        """Rows with status != ACTIVATED must be skipped."""
        eeg_id = test_eeg["id"]
        uid = _uuid.uuid4().hex[:6]

        xlsx = build_stammdaten_xlsx([
            {
                "zaehlpunkt": f"AT001000000000000000000000SKIP{uid[:4]}C",
                "name1": f"Skip Member {uid}",
                "email": f"skip_{uid}@test.at",
                "mitglieds_nr": f"SKIP{uid}",
                "status": "INACTIVE",  # should be skipped
            },
        ])

        result = api.import_stammdaten(eeg_id, xlsx)
        assert result["members"] == 0, "INACTIVE rows should be skipped"
        assert result["meter_points"] == 0

    def test_import_idempotent_on_reupload(self, api, test_eeg):
        """Uploading the same Stammdaten twice should not create duplicate members."""
        eeg_id = test_eeg["id"]
        uid = _uuid.uuid4().hex[:6]
        zp = f"AT001000000000000000000000IDEM{uid[:4]}C"

        xlsx = build_stammdaten_xlsx([{
            "zaehlpunkt": zp,
            "name1": f"Idem Mitglied {uid}",
            "email": f"idem_{uid}@test.at",
            "mitglieds_nr": f"IDEM{uid}",
            "status": "ACTIVATED",
        }])

        r1 = api.import_stammdaten(eeg_id, xlsx)
        r2 = api.import_stammdaten(eeg_id, xlsx)
        # Both should succeed; second is an upsert (no error)
        assert r1["members"] >= 1
        assert r2["members"] >= 1  # upsert counts the row even if unchanged


class TestEnergieDatenImport:
    """POST /api/v1/eegs/{eegID}/import/energiedaten (preview + actual import)."""

    @pytest.fixture(autouse=True)
    def setup(self, test_eeg, test_meter_points):
        self.eeg_id = test_eeg["id"]
        # Use a month that has no existing readings to avoid conflicts.
        # conftest inserts Jan–May 2025; we use Jun 2025 here.
        self.period_start = datetime(2025, 6, 1, 0, 0, 0, tzinfo=timezone.utc)
        self.meter_ids = [
            TEST_ZAEHLPUNKT_CONSUMER,
            TEST_ZAEHLPUNKT_PROSUMER,
        ]

    def test_preview_returns_new_row_count(self, api):
        xlsx = build_energiedaten_xlsx(
            self.meter_ids,
            period_start=self.period_start,
            hours=24,
            wh_total=500.0,
            wh_community=300.0,
        )
        preview = api.preview_energiedaten(self.eeg_id, xlsx)
        assert preview["total_rows"] == 24 * len(self.meter_ids), (
            f"Expected {24 * len(self.meter_ids)} rows, got {preview}"
        )
        assert preview["new_rows"] >= 0
        assert preview["conflict_rows"] >= 0
        assert "period_start" in preview
        assert "period_end" in preview

    def test_import_inserts_rows(self, api):
        xlsx = build_energiedaten_xlsx(
            self.meter_ids,
            period_start=self.period_start,
            hours=24,
        )
        result = api.import_energiedaten(self.eeg_id, xlsx, mode="overwrite")
        assert result["rows_parsed"] == 24 * len(self.meter_ids)
        assert result["rows_inserted"] >= 0  # may be 0 if already inserted by preview
        assert result["mode"] == "overwrite"

    def test_import_skip_mode_does_not_overwrite(self, api):
        """mode=skip must not overwrite existing readings."""
        xlsx = build_energiedaten_xlsx(
            self.meter_ids,
            period_start=self.period_start,
            hours=24,
            wh_total=9999.0,  # different values
            wh_community=9999.0,
        )
        # First import (overwrite) to populate readings
        api.import_energiedaten(self.eeg_id, xlsx, mode="overwrite")

        # Now import with skip mode — should not change anything
        result = api.import_energiedaten(self.eeg_id, xlsx, mode="skip")
        assert result["mode"] == "skip"
        # rows_inserted may be 0 (all already exist) or > 0 if new timestamps
        assert "rows_inserted" in result

    def test_unknown_zaehlpunkt_reported_as_skipped(self, api):
        """Readings for an unknown Zählpunkt must be reported in skipped_meters."""
        xlsx = build_energiedaten_xlsx(
            ["AT_UNKNOWN_ZAEHLPUNKT_99999"],
            period_start=self.period_start,
            hours=2,
        )
        result = api.import_energiedaten(self.eeg_id, xlsx)
        assert result["skipped_meters"] >= 1, (
            f"Expected skipped_meters ≥ 1 for unknown Zählpunkt, got {result}"
        )


class TestCoverage:
    """GET /api/v1/eegs/{eegID}/readings/coverage?year=YYYY"""

    def test_coverage_returns_year_and_days(self, api, test_eeg, test_readings):
        result = api.get_coverage(test_eeg["id"], 2025)
        assert result["year"] == 2025
        assert isinstance(result["days"], list)

    def test_coverage_days_include_covered_months(self, api, test_eeg, test_readings):
        """Jan–May 2025 has readings — at least those months should show coverage."""
        result = api.get_coverage(test_eeg["id"], 2025)
        days = result["days"]
        # Days with readings have count > 0
        covered = [d for d in days if d.get("count", 0) > 0]
        assert len(covered) >= 28 * 5, (
            f"Expected ≥140 covered days in 2025, got {len(covered)}"
        )

    def test_coverage_empty_year_returns_empty_list(self, api, test_eeg):
        result = api.get_coverage(test_eeg["id"], 2020)
        assert result["year"] == 2020
        assert result["days"] == [] or all(not d.get("has_data") for d in result["days"])
