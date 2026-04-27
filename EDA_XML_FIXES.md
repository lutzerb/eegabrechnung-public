# EDA XML Korrekturen — Arbeitsplan

Recherchiert am 2026-04-09 gegen ebutilities.at

---

## Status-Übersicht

| # | Aufgabe | Status |
|---|---------|--------|
| 1 | ECMPList-Builder (ANFORDERUNG_CPF / ECC) | ✅ Erledigt |
| 2 | CMRequest-Builder 01.21 + ProcessDate/MeteringPoint-Namespace | ✅ Erledigt |
| 3 | EC_EINZEL_ABM → ECMPList ANFORDERUNG_ECC — Namespace-Fix (cp: statt ct:) | ✅ Erledigt (2026-04-10) |
| 4 | DocumentCreationDateTime Timezone-Fix (+01:00/+02:00) | ✅ Erledigt |
| 4b | EDASendError Parser + handleEDASendError in worker.go | ✅ Erledigt |
| 4c | eda_messages.eda_process_id FK (Migration 042) | ✅ Erledigt |
| 5 | CMRequest 01.21 → 01.30 (aktiv ab 12.04.2026) | ✅ Code fertig, aktiv ab 12.04 |
| 5b | IMAP DialTLS context timeout + PEEK-Fix | ✅ Erledigt (2026-04-10) |
| 6 | EC_REQ_OFF (Offline-Anmeldung, CMRequest 01.30) | ✅ Code fertig, aktiv ab 12.04 |
| 7 | EC_PODLIST (Zählpunktliste anfordern) | ✅ Erledigt + getestet (2026-04-10) |
| 8 | CM_REV_SP (Widerruf durch EEG) | ✅ Code fertig, ungetestet |
| 9 | CM_REV_CUS (Widerruf durch Kunden — inbound) | ✅ Code fertig, ungetestet |

### Bekannte Infrastruktur-Probleme (behoben)
- **Test-Worker stiehlt Jobs**: `eegabrechnung-eda-worker-test` nutzt dieselbe DB wie MAIL-Worker.
  Bei gleichzeitigem Betrieb verarbeitet FILE-Transport Jobs die eigentlich per SMTP gesendet werden müssten.
  → **Fix**: Test-Worker stoppen wenn MAIL-Worker aktiv ist (`docker stop eegabrechnung-eegabrechnung-eda-worker-test-1`)
- **BODY[] vs BODY.PEEK[]**: IMAP-Fetch ohne PEEK markiert Mails automatisch als Seen auf dem Server.
  → **Fix**: `{Peek: true}` in FetchItemBodySection (commit d327f73)

---

## Schritt 5 (KRITISCH): CMRequest 01.21 → 01.30

**Gültig ab: 12.04.2026** — d.h. in 2-3 Tagen!

### Änderungen:

**`api/internal/eda/xml/cmrequest_builder.go`:**
- `cmRequestNS`: `http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p21`
  → `http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p30`
- `cmRequestSchemaVer`: `"01.21"` → `"01.30"`
- Neues optionales Feld `Purpose` (string, max 35 chars) in CMRequestParams + cmInnerRequestXML

**`api/internal/eda/transport/mail.go` (edanetProzessID map):**
- `"EC_REQ_ONL": "EC_REQ_ONL_02.10"` → `"EC_REQ_ONL_02.30"`
- `"EC_EINZEL_ANM": "EC_REQ_ONL_02.10"` → `"EC_REQ_ONL_02.30"`

### CMRequest 01.30 XML Beispiel:
```xml
<cp:CMRequest xmlns:cp="http://www.ebutilities.at/schemata/customerconsent/cmrequest/01p30"
              xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
              xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.30">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber"><ct:MessageAddress>AT0030...</ct:MessageAddress></ct:Sender>
      <ct:Receiver AddressType="ECNumber"><ct:MessageAddress>AT0020...</ct:MessageAddress></ct:Receiver>
      <ct:DocumentCreationDateTime>2026-04-12T10:00:00+02:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>ANFORDERUNG_ECON</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>ABC123...</ct:MessageId>
    <ct:ConversationId>DEF456...</ct:ConversationId>
    <cp:ProcessDate>2026-04-12</cp:ProcessDate>
    <cp:MeteringPoint>AT0010000000000000000000000012345</cp:MeteringPoint>
    <cp:CMRequestId>GHI789...</cp:CMRequestId>
    <cp:CMRequest>
      <cp:ReqDatType>ECON</cp:ReqDatType>
      <cp:DateFrom>2026-05-01</cp:DateFrom>
      <cp:ECID>AT0030000000000000000000EEG00001</cp:ECID>
      <cp:ECPartFact>100</cp:ECPartFact>
      <cp:EnergyDirection>CONSUMPTION</cp:EnergyDirection>
    </cp:CMRequest>
  </cp:ProcessDirectory>
</cp:CMRequest>
```

---

## Schritt 6: EC_REQ_OFF (Offline-Anmeldeanforderung)

Prozess: `EC_REQ_OFF (02.20)` — für Kunden ohne Smart Meter / ohne Online-Zustimmung

### Unterschiede zu EC_REQ_ONL:
- Selber CMRequest 01.30 Builder
- MessageCode: `ANFORDERUNG_ECOF` (statt `ANFORDERUNG_ECON`)
- Email-Subject Prozess-ID: `EC_REQ_OFF_02.20`
- Flow: EEG → NB → Papierdokument zum Kunden → NB bestätigt

### Zu implementieren:
1. `mail.go`: `"EC_REQ_OFF": "EC_REQ_OFF_02.20"` hinzufügen
2. `handler/eda.go`: `AnmeldungOffline` Handler (POST `/eda/anmeldung-offline`)
   - Identisch zu `AnmeldungOnline` aber MessageCode `ANFORDERUNG_ECOF`
   - ProcessType `"EC_REQ_OFF"` in DB
3. `api/cmd/server/main.go`: Route registrieren
4. `web/app/api/eegs/[eegId]/eda/anmeldung-offline/route.ts`: Proxy-Route
5. `web/components/eda-action-forms.tsx`: Tab "Offline-Anmeldung" hinzufügen

---

## Schritt 7: EC_PODLIST (Zählpunktliste)

Prozess: `EC_PODLIST (01.00)` — EEG fordert aktuelle Zählpunktliste vom NB an

### Technisch:
- Schema: CPRequest 01.12 (wie EC_REQ_PT), aber MessageCode `ANFORDERUNG_ECP`
- MeteringPoint im ProcessDirectory = ECID (Gemeinschafts-ID!)
- Keine DateTimeFrom/DateTimeTo Extension nötig
- Inbound Response: ECMPList `SENDEN_ECP` (bestehender Parser)
- Email-Subject: `EC_PODLIST_01.00`

### Zu implementieren:
1. `cprequest_builder.go`: `BuildPODList` Funktion (CPRequest 01.12, `ANFORDERUNG_ECP`, kein Extension)
2. `mail.go`: `"EC_PODLIST": "EC_PODLIST_01.00"`
3. `handler/eda.go`: `PODList` Handler (POST `/eda/podlist`)
4. `api/cmd/server/main.go`: Route registrieren
5. `web/app/api/eegs/[eegId]/eda/podlist/route.ts`
6. `web/components/eda-action-forms.tsx`: Button "Zählpunktliste anfordern"

---

## Schritt 8: CM_REV_SP (Widerruf durch EEG)

Prozess: `CM_REV_SP (01.00)` — EEG widerruft Zustimmung für einen Zählpunkt

### Schema: CMRevoke 01.10
```
Namespace: http://www.ebutilities.at/schemata/customerconsent/cmrevoke/01p10
XSD: CMRevoke_01p10.xsd
MessageCode: AUFHEBUNG_CCMS
Email-Subject: CM_REV_SP_01.00
```

### Felder:
- `ConsentId` (string): ConversationId des ursprünglichen EC_REQ_ONL Prozesses
- `MeteringPoint` (string): Zählpunkt
- `ConsentEnd` (date): Enddatum der Zustimmung
- `ReasonKey` (int, 0-9, optional): Grund-Code
- `Reason` (string, max 50, optional): Freitext

### Zu implementieren:
1. `api/internal/eda/xml/cmrevoke_builder.go` (neue Datei)
2. `mail.go`: `"CM_REV_SP": "CM_REV_SP_01.00"`
3. `handler/eda.go`: `WiderrufEEG` Handler (POST `/eda/widerruf`)
4. DB: Kein neues Feld nötig (nutzt existierende EDAProcess-Felder)
5. `api/cmd/server/main.go`: Route registrieren
6. `web/app/api/eegs/[eegId]/eda/widerruf/route.ts`
7. `web/components/eda-action-forms.tsx`: Tab "Widerruf"

### CMRevoke 01.10 XML Beispiel:
```xml
<cp:CMRevoke xmlns:cp="http://www.ebutilities.at/schemata/customerconsent/cmrevoke/01p10"
             xmlns:ct="http://www.ebutilities.at/schemata/customerprocesses/common/types/01p20"
             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <cp:MarketParticipantDirectory DocumentMode="PROD" Duplicate="false" SchemaVersion="01.10">
    <ct:RoutingHeader>
      <ct:Sender AddressType="ECNumber"><ct:MessageAddress>AT0030...</ct:MessageAddress></ct:Sender>
      <ct:Receiver AddressType="ECNumber"><ct:MessageAddress>AT0020...</ct:MessageAddress></ct:Receiver>
      <ct:DocumentCreationDateTime>2026-04-12T10:00:00+02:00</ct:DocumentCreationDateTime>
    </ct:RoutingHeader>
    <ct:Sector>01</ct:Sector>
    <cp:MessageCode>AUFHEBUNG_CCMS</cp:MessageCode>
  </cp:MarketParticipantDirectory>
  <cp:ProcessDirectory>
    <ct:MessageId>MSG123</ct:MessageId>
    <ct:ConversationId>CONV456</ct:ConversationId>
    <cp:ProcessDate>2026-04-12</cp:ProcessDate>
    <cp:MeteringPoint>AT0010000000000000000000000012345</cp:MeteringPoint>
    <cp:ConsentId>ORIGCONV123</cp:ConsentId>
    <cp:ConsentEnd>2026-04-30</cp:ConsentEnd>
    <cp:ReasonKey>1</cp:ReasonKey>            <!-- optional -->
    <cp:Reason>Mitglied ausgetreten</cp:Reason> <!-- optional, max 50 chars -->
  </cp:ProcessDirectory>
</cp:CMRevoke>
```

---

## Schritt 9: CM_REV_CUS (inbound — Widerruf durch Kunden)

Prozess: Netzbetreiber schickt uns `CM_REV_CUS` wenn Kunde selbst widerruft

### Zu implementieren:
1. `api/internal/eda/xml/cmrevoke_parser.go` (neue Datei): `IsCMRevoke(xml)`, `ParseCMRevoke(xml)` → fields: ConversationId, MeteringPoint, ConsentId, ConsentEnd
2. `api/internal/eda/worker.go`: Inbound-case für CMRevoke → Prozess auf `completed` setzen + error_msg "Kunde hat Zustimmung widerrufen"
3. `api/internal/eda/transport/mail.go`: CMRevoke-Erkennung in xmlSearchString (marker `<cp:CMRevoke`)

---

## Schlüssel-Dateien

| Datei | Zweck |
|-------|-------|
| `api/internal/eda/xml/cmrequest_builder.go` | CMRequest Builder (→ 01.30 upgraden) |
| `api/internal/eda/xml/ecmplist_builder.go` | ECMPList Builder (korrekt) |
| `api/internal/eda/xml/cprequest_builder.go` | CPRequest Builder (01.12 korrekt) |
| `api/internal/eda/transport/mail.go` | Email-Subject-Mapping (edanetProzessID) |
| `api/internal/handler/eda.go` | HTTP-Handler für alle EDA-Prozesse |
| `api/cmd/server/main.go` | Route-Registrierung |
| `api/internal/eda/worker.go` | Inbound-Message-Routing |
| `web/components/eda-action-forms.tsx` | EDA UI-Formulare |
