"""Mailpit HTTP API client and SMTP injection helper."""
import smtplib
import time
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
import requests


class MailpitClient:
    """Thin client for the Mailpit REST API (port 8025)."""

    def __init__(self, base_url: str = "http://localhost:8025"):
        self.base_url = base_url.rstrip("/")

    def list_messages(self) -> list[dict]:
        resp = requests.get(f"{self.base_url}/api/v1/messages")
        resp.raise_for_status()
        return resp.json().get("messages", [])

    def get_message(self, message_id: str) -> dict:
        resp = requests.get(f"{self.base_url}/api/v1/message/{message_id}")
        resp.raise_for_status()
        return resp.json()

    def get_message_raw(self, message_id: str) -> str:
        resp = requests.get(f"{self.base_url}/api/v1/message/{message_id}/raw")
        resp.raise_for_status()
        return resp.text

    def delete_all(self) -> None:
        """Delete all messages (clean state before a test)."""
        requests.delete(f"{self.base_url}/api/v1/messages")

    def wait_for_message(
        self, subject_contains: str = "", timeout: float = 10.0, poll: float = 0.5
    ) -> dict:
        """Poll until a message matching subject_contains appears, or timeout."""
        deadline = time.time() + timeout
        while time.time() < deadline:
            for msg in self.list_messages():
                subj = msg.get("Subject", "")
                if not subject_contains or subject_contains in subj:
                    return msg
            time.sleep(poll)
        raise TimeoutError(
            f"No message with subject containing {subject_contains!r} "
            f"arrived within {timeout}s"
        )

    def find_messages_to(self, to_addr: str) -> list[dict]:
        return [
            m
            for m in self.list_messages()
            if any(r.get("Address") == to_addr for r in m.get("To", []))
        ]


def smtp_send(
    smtp_host: str,
    smtp_port: int,
    from_addr: str,
    to_addr: str,
    subject: str,
    xml_body: str,
) -> None:
    """Send an XML EDA message via plain SMTP (no auth, no TLS)."""
    msg = MIMEMultipart()
    msg["From"] = from_addr
    msg["To"] = to_addr
    msg["Subject"] = subject

    xml_part = MIMEText(xml_body, "xml", "utf-8")
    xml_part.add_header("Content-Disposition", 'attachment; filename="mako.xml"')
    msg.attach(xml_part)

    with smtplib.SMTP(smtp_host, smtp_port) as s:
        s.sendmail(from_addr, [to_addr], msg.as_bytes())
