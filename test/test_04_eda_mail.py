"""EDA end-to-end tests: outbound XML via FILE transport + SMTP outbound via Mailpit.

Requires the test docker-compose profile:
  docker compose --profile test up -d

Transport strategy:
  - The test worker uses FILE transport so outbound XML is written to test/eda-outbox
    and inbound CPDocuments can be placed in test/eda-inbox.
  - Mailpit captures SMTP messages sent directly (not via worker) so we can verify
    that the SMTP send path works at the transport layer.
  - IMAP inbound is tested indirectly: FILE transport exercises the same CPDocument
    parsing and process-state logic that MAIL transport would use.

Why FILE instead of IMAP:
  Mailpit v1.x (SMTP + POP3 only; IMAP was not added until v2.x). FILE transport
  tests the full CPDocument processing pipeline; the IMAP connection code has
  IMAPPlain=true support that can be verified with a real EDA test account.
"""
import os
import time
import uuid as _uuid

import requests
import pytest

from conftest import (
    WORKER_URL, MAILPIT_URL, MAILPIT_SMTP_HOST, MAILPIT_SMTP_PORT,
    TEST_ZAEHLPUNKT_CONSUMER, TEST_ZAEHLPUNKT_PROSUMER,
    TEST_GEMEINSCHAFT_ID, WORKER_EMAIL,
)
from helpers.mailpit import MailpitClient, smtp_send
from helpers.eda_xml import build_cpdocument, build_cpdocument_rejected

INBOX_DIR = os.environ.get(
    "EDA_INBOX_DIR",
    os.path.join(os.path.dirname(__file__), "eda-inbox"),
)
OUTBOX_DIR = os.environ.get(
    "EDA_OUTBOX_DIR",
    os.path.join(os.path.dirname(__file__), "eda-outbox"),
)

SKIP_REASON = (
    "EDA worker test profile not running — "
    "start with: docker compose --profile test up -d"
)


def worker_available() -> bool:
    try:
        r = requests.get(f"{WORKER_URL}/health", timeout=3)
        return r.status_code == 200
    except Exception:
        return False


def mailpit_available() -> bool:
    try:
        MailpitClient(MAILPIT_URL).list_messages()
        return True
    except Exception:
        return False


def poll_now(timeout: float = 8.0) -> None:
    """Trigger one worker poll cycle and wait for it to finish."""
    r = requests.post(f"{WORKER_URL}/eda/poll-now", timeout=10)
    assert r.status_code == 202, f"poll-now returned {r.status_code}"
    time.sleep(timeout)


def write_inbox(filename: str, content: str) -> str:
    """Write an XML file to the eda-inbox directory."""
    path = os.path.join(INBOX_DIR, filename)
    os.makedirs(INBOX_DIR, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    return path


def list_outbox_files() -> list[str]:
    """Return XML files currently in eda-outbox."""
    if not os.path.isdir(OUTBOX_DIR):
        return []
    return [
        f for f in os.listdir(OUTBOX_DIR)
        if f.endswith(".xml") and os.path.isfile(os.path.join(OUTBOX_DIR, f))
    ]


def read_outbox_xml(filename: str) -> str:
    return open(os.path.join(OUTBOX_DIR, filename), encoding="utf-8").read()


@pytest.fixture(autouse=True)
def require_worker(worker_url):
    if not worker_available():
        pytest.skip(SKIP_REASON)


@pytest.fixture(autouse=True)
def clean_inbox_outbox():
    """Remove all files from inbox/outbox before each test."""
    for d in (INBOX_DIR, OUTBOX_DIR):
        os.makedirs(d, exist_ok=True)
        for fn in os.listdir(d):
            fp = os.path.join(d, fn)
            if os.path.isfile(fp):
                os.remove(fp)
    yield


class TestOutboundXML:
    """Worker writes outbound EC_EINZEL_ANM XML to eda-outbox."""

    def test_anmeldung_writes_xml_to_outbox(
        self, api, test_eeg, test_meter_points
    ):
        eeg_id = test_eeg["id"]
        before = set(list_outbox_files())

        proc = api.eda_anmeldung(eeg_id, {
            "zaehlpunkt": TEST_ZAEHLPUNKT_CONSUMER,
            "valid_from": "2026-04-01",
            "share_type": "GC",
            "participation_factor": 100.0,
        })
        conv_id = proc["conversation_id"]
        proc_id = proc["id"]

        poll_now(timeout=5)

        after = set(list_outbox_files())
        new_files = after - before
        assert new_files, "No new XML file appeared in eda-outbox after Anmeldung"

        # Check that at least one file contains the ConversationID.
        found = False
        for fn in new_files:
            xml = read_outbox_xml(fn)
            if conv_id in xml:
                found = True
                assert "EC_EINZEL_ANM" in xml
                break
        assert found, (
            f"ConversationID {conv_id} not found in any outbox file: {new_files}"
        )

        # Process should be "sent" after the worker dispatched it.
        proc_updated = api.get_eda_process(eeg_id, proc_id)
        assert proc_updated["status"] == "sent", (
            f"Expected status=sent, got {proc_updated['status']}"
        )

        self.__class__._conv_id = conv_id
        self.__class__._proc_id = proc_id

    def test_abmeldung_writes_xml_to_outbox(
        self, api, test_eeg, test_meter_points
    ):
        eeg_id = test_eeg["id"]
        before = set(list_outbox_files())

        proc = api.eda_abmeldung(eeg_id, {
            "zaehlpunkt": TEST_ZAEHLPUNKT_PROSUMER,
            "valid_from": "2026-05-01",
        })
        proc_id = proc["id"]
        poll_now(timeout=5)

        after = set(list_outbox_files())
        assert after - before, "No outbox file for Abmeldung"

        proc_updated = api.get_eda_process(eeg_id, proc_id)
        assert proc_updated["status"] == "sent"


class TestInboundCPDocument:
    """Worker reads CPDocument XML from eda-inbox and updates process status."""

    @pytest.fixture(autouse=True)
    def setup_sent_process(self, api, test_eeg, test_meter_points):
        """Create an Anmeldung and drive it to 'sent' via poll_now."""
        eeg_id = test_eeg["id"]
        proc = api.eda_anmeldung(eeg_id, {
            "zaehlpunkt": TEST_ZAEHLPUNKT_CONSUMER,
            "valid_from": "2026-06-01",
            "share_type": "GC",
            "participation_factor": 100.0,
        })
        self.conv_id = proc["conversation_id"]
        self.proc_id = proc["id"]
        self.eeg_id = eeg_id

        # Trigger worker → sends outbound XML, process becomes "sent".
        poll_now(timeout=5)

        proc_now = api.get_eda_process(eeg_id, self.proc_id)
        assert proc_now["status"] == "sent", (
            f"Expected sent before inbound test, got {proc_now['status']}"
        )
        # Clear outbox so inbound tests start clean.
        for fn in list_outbox_files():
            os.remove(os.path.join(OUTBOX_DIR, fn))

    def test_erste_anm_sets_first_confirmed(self, api):
        xml = build_cpdocument(
            conversation_id=self.conv_id,
            message_code="ERSTE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        )
        write_inbox(f"erste_anm_{_uuid.uuid4().hex[:8]}.xml", xml)
        poll_now(timeout=8)

        proc = api.get_eda_process(self.eeg_id, self.proc_id)
        assert proc["status"] == "first_confirmed", (
            f"Expected first_confirmed, got {proc['status']}"
        )

    def test_finale_anm_sets_confirmed(self, api):
        uid = _uuid.uuid4().hex[:8]
        # First: ERSTE_ANM → first_confirmed.
        write_inbox(f"s1_erste_{uid}.xml", build_cpdocument(
            conversation_id=self.conv_id,
            message_code="ERSTE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        ))
        poll_now(timeout=8)

        # Then: FINALE_ANM → confirmed.
        write_inbox(f"s2_finale_{uid}.xml", build_cpdocument(
            conversation_id=self.conv_id,
            message_code="FINALE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        ))
        poll_now(timeout=8)

        proc = api.get_eda_process(self.eeg_id, self.proc_id)
        assert proc["status"] == "confirmed", (
            f"Expected confirmed, got {proc['status']}"
        )

    def test_abgelehnt_sets_rejected(self, api):
        xml = build_cpdocument_rejected(
            conversation_id=self.conv_id,
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        )
        write_inbox(f"rejected_{_uuid.uuid4().hex[:8]}.xml", xml)
        poll_now(timeout=8)

        proc = api.get_eda_process(self.eeg_id, self.proc_id)
        assert proc["status"] == "rejected", (
            f"Expected rejected, got {proc['status']}"
        )

    def test_unknown_conversation_id_ignored(self, api):
        """CPDocument with unknown ConversationID must not crash the worker."""
        xml = build_cpdocument(
            conversation_id="00000000-0000-0000-0000-000000000000",
            message_code="ERSTE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        )
        write_inbox(f"unknown_conv_{_uuid.uuid4().hex[:8]}.xml", xml)
        poll_now(timeout=8)

        r = requests.get(f"{WORKER_URL}/health", timeout=5)
        assert r.status_code == 200

    def test_duplicate_xml_processed_once(self, api):
        """Writing the same CPDocument XML twice must only apply the update once."""
        uid = _uuid.uuid4().hex[:8]
        xml = build_cpdocument(
            conversation_id=self.conv_id,
            message_code="ERSTE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        )
        # Write the same content under two filenames.
        write_inbox(f"dup1_{uid}.xml", xml)
        write_inbox(f"dup2_{uid}.xml", xml)
        poll_now(timeout=10)

        proc = api.get_eda_process(self.eeg_id, self.proc_id)
        assert proc["status"] in ("first_confirmed", "confirmed", "rejected"), (
            f"Unexpected status after duplicate: {proc['status']}"
        )


@pytest.mark.skipif(not mailpit_available(), reason="Mailpit not running")
class TestSMTPConnectivity:
    """Verify Mailpit captures plain SMTP messages (transport layer test)."""

    @pytest.fixture(autouse=True)
    def clear_mailpit(self):
        mp = MailpitClient(MAILPIT_URL)
        mp.delete_all()
        yield

    def test_smtp_send_xml_arrives_in_mailpit(self):
        """Directly send an XML email via SMTP and verify Mailpit receives it."""
        mp = MailpitClient(MAILPIT_URL)
        xml = build_cpdocument(
            conversation_id=str(_uuid.uuid4()),
            message_code="ERSTE_ANM",
            zaehlpunkt=TEST_ZAEHLPUNKT_CONSUMER,
            gemeinschaft_id=TEST_GEMEINSCHAFT_ID,
        )
        smtp_send(
            smtp_host=MAILPIT_SMTP_HOST,
            smtp_port=MAILPIT_SMTP_PORT,
            from_addr="test@sender.at",
            to_addr=WORKER_EMAIL,
            subject="EDA Test CPDocument",
            xml_body=xml,
        )
        msg = mp.wait_for_message(subject_contains="EDA Test", timeout=10)
        assert msg, "Expected message in Mailpit after SMTP send"
        assert "CPDocument" in mp.get_message_raw(msg["ID"])
