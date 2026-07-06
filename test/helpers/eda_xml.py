"""Builders for EDA CPDocument XML (inbound confirmation messages from Netzbetreiber)."""
import uuid
from datetime import datetime


def build_cpdocument(
    conversation_id: str,
    message_code: str,
    zaehlpunkt: str,
    gemeinschaft_id: str,
    sender: str = "AT001000000000000000000000000000001",
    receiver: str = "AT001000000000000000000000000000002",
    valid_from: str | None = None,
) -> str:
    """Build a CPDocument XML confirmation message.

    message_code options:
      ERSTE_ANM       → first_confirmed (Anmeldung first step)
      FINALE_ANM      → confirmed (Anmeldung final)
      ABSCHLUSS_ECON  → confirmed (Abmeldung/Teilnahmefaktor complete)
      ABGELEHNT_ANM   → rejected
    """
    msg_id = str(uuid.uuid4())
    now = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S")
    today = datetime.utcnow().strftime("%Y-%m-%d")
    vf = valid_from or today

    return f"""<?xml version="1.0" encoding="UTF-8"?>
<CPDocument
    DocumentMode="PROD"
    SchemaVersion="01.40"
    xmlns="http://www.ebutilities.at/schemata/customerprocesses/cpdocument/01p40"
    xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <MarketParticipantDirectory DocumentMode="PROD" SchemaVersion="01.40">
    <MessageCode>{message_code}</MessageCode>
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber">
        <ct:MessageAddress>{sender}</ct:MessageAddress>
      </ct:Sender>
      <ct:Receiver AddressType="ECNumber">
        <ct:MessageAddress>{receiver}</ct:MessageAddress>
      </ct:Receiver>
      <ct:DocumentCreationDateTime>{now}</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
  </MarketParticipantDirectory>
  <ProcessDirectory>
    <ct:MessageId>{msg_id}</ct:MessageId>
    <ct:ConversationId>{conversation_id}</ct:ConversationId>
    <ct:ProcessDate>{today}</ct:ProcessDate>
    <ct:MeteringPoint>{zaehlpunkt}</ct:MeteringPoint>
    <CommunityID>{gemeinschaft_id}</CommunityID>
    <ValidFrom>{vf}</ValidFrom>
  </ProcessDirectory>
</CPDocument>"""


def build_cpdocument_rejected(
    conversation_id: str,
    zaehlpunkt: str,
    gemeinschaft_id: str,
    reason: str = "ABGELEHNT_ANM",
) -> str:
    return build_cpdocument(
        conversation_id=conversation_id,
        message_code=reason,
        zaehlpunkt=zaehlpunkt,
        gemeinschaft_id=gemeinschaft_id,
    )
