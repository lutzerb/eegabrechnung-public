# 4 Mitglieder & Zählpunkte

Mitglieder sind die Teilnehmer einer Energiegemeinschaft. Jedes Mitglied besitzt einen oder mehrere Zählpunkte, über die Verbrauchs- bzw. Einspeisenergiedaten erfasst werden.

![Mitgliederliste](screenshots/members-list.png)

---

## 4.1 Mitgliedertypen

| Typ | Energierichtung | Abrechnung |
|-----|----------------|------------|
| `CONSUMER` | Nur Verbrauch | Erhält Rechnung |
| `PROSUMER` | Verbrauch und Einspeisung | Erhält Rechnung (Nettobetrag nach Verrechnung) |
| `PRODUCER` | Nur Einspeisung | Erhält Gutschrift (kein Rechnungsdokument) |

<div class="tip">
Der Typ <em>PRODUCER</em> löst automatisch eine Gutschrift statt einer Rechnung aus — vorausgesetzt, Gutschriften sind in den EEG-Einstellungen aktiviert (<code>generate_credit_notes</code>). Ist die Option deaktiviert, wird auch für Produzenten ein normales Rechnungsdokument erzeugt.
</div>

---

## 4.2 Mitglied anlegen

Neues Mitglied unter `/eegs/{eegId}/members/new` anlegen. Pflichtfeld ist ausschließlich **Vorname / Name** (`name1`).

### Stammdaten

| Feld | Pflicht | Hinweis |
|------|---------|---------|
| Vorname / Name (`name1`) | ja | Natürliche Person: Vorname; Firma: vollständiger Firmenname |
| Nachname (`name2`) | nein | Wird für natürliche Personen zusätzlich zu `name1` angegeben |
| Mitgliedsnummer | nein | Bei leer gelassenem Feld automatisch vergeben |
| E-Mail | nein | Wird für Onboarding-Benachrichtigungen und Mitglieder-Portal verwendet |
| IBAN | nein | Bankverbindung des Mitglieds für SEPA-Zahlungen |
| Straße | nein | Adresse des Mitglieds |
| PLZ | nein | |
| Ort | nein | |

### Unternehmensart (`business_role`)

Die Unternehmensart steuert das steuerrechtliche Standardverhalten bei Einspeisvergütungen:

| Wert | Bedeutung |
|------|-----------|
| `privat` | Privatperson — keine USt auf Gutschriften |
| `kleinunternehmer` | Kleinunternehmer gem. § 6 UStG — keine USt |
| `landwirt_pauschaliert` | Landwirt pauschaliert gem. § 22 UStG — 13 % USt |
| `landwirt` | Landwirt buchführungspflichtig — Regelbesteuerung |
| `unternehmen` | Unternehmen — Regelbesteuerung (20 %) |
| `gemeinde_bga` | Gemeinde (Betrieb gewerblicher Art) — Regelbesteuerung |
| `gemeinde_hoheitlich` | Gemeinde (hoheitlicher Bereich) — keine USt |

### MwSt-Überschreibung (per Mitglied)

Abweichend von der automatischen Logik laut Unternehmensart kann die Umsatzsteuer manuell übersteuert werden:

| Einstellung | Bedeutung |
|-------------|-----------|
| Automatisch (laut Unternehmensart) | Systemlogik bestimmt `use_vat` / `vat_pct` |
| USt-pflichtig (manuell) | `use_vat = true`; `vat_pct` frei wählbar |
| Nicht USt-pflichtig (manuell) | `use_vat = false`, `vat_pct = 0` |

<div class="tip">
Die MwSt-Überschreibung gilt ausschließlich für <strong>Einspeisvergütungen (Gutschriften)</strong> — nicht für Verbrauchsrechnungen. Letztere folgen immer der EEG-weiten USt-Einstellung.
</div>

### Österreichische USt-Matrix für Gutschriften

| Mitgliedstyp / Konstellation | `use_vat` | `vat_pct` | Anmerkung |
|-----------------------------|-----------|-----------|-----------|
| Privatperson, Kleinunternehmer | `false` | 0 | Steuerbefreit |
| Landwirt (pauschaliert, § 22) | `true` | 13 | Vorsteuerpauschale |
| Unternehmen ohne UID-Nummer | `true` | 20 | Regelbesteuerung |
| Unternehmen **mit** UID-Nummer | `false` | 0 | Reverse Charge gem. § 19 UStG |
| Split Bezug / Einspeisung (PROSUMER) | — | — | Getrennte Positionen in der Rechnung |

<div class="warning">
Wird die UID-Nummer eines Mitglieds nachträglich eingetragen oder entfernt, ändert sich das Steuerregime für künftige Gutschriften. Bereits ausgestellte Dokumente werden nicht rückwirkend angepasst.
</div>

### Mitgliedsdaten ergänzen

| Feld | Zweck |
|------|-------|
| UID-Nummer | Reverse Charge auf Gutschriften (§ 19 UStG), erscheint auf Rechnung |
| Beitrittsdatum | Beginn der Mitgliedschaft — wird bei Abrechnungsperioden berücksichtigt |
| Austrittsdatum | Ende der Mitgliedschaft |
| Status (`ACTIVE` / `INACTIVE`) | INACTIVE-Mitglieder werden bei Abrechnungsläufen übersprungen |

### Mitgliederstatus-Logik

- **ACTIVE**: Mitglied wird normal abgerechnet.
- **INACTIVE**: Wird bei Abrechnungsläufen übersprungen, es sei denn, der Lauf wird mit der Option „Erzwingen" gestartet.
- Beitritts- und Austrittsdatum schränken den abzurechnenden Zeitraum ein: Vor dem Beitrittsdatum und nach dem Austrittsdatum werden keine Verbrauchsmengen in Rechnung gestellt.

---

## 4.3 Zählpunkte (Meter Points)

Zählpunkte werden innerhalb einer Mitgliederdetailseite unter `/eegs/{eegId}/members/{memberId}/meter-points/new` angelegt.

### Stammdaten eines Zählpunkts

| Feld | Pflicht | Format / Werte |
|------|---------|---------------|
| Zählpunkt-ID | ja | 33-stellig gemäß österreichischem Standard, z.B. `AT0010000000000000001000000000001` |
| Energierichtung | ja | `CONSUMPTION` (Bezug) oder `GENERATION` (Einspeisung) |
| Einspeisungsart (`generation_type`) | nein | `PV`, `Windkraft`, `Wasserkraft`, `Biomasse`, `Sonstige` |
| Verteilungsmodell | nein | Aktuell nur `DYNAMIC` (dynamisch) produktiv unterstützt |
| Zugeteilte Menge (%) | nein | Für statisches Verteilungsmodell vorgesehen, derzeit aber nicht produktiv implementiert |
| Status | nein | `NEW` (Neu) oder `ACTIVATED` (Aktiviert) |
| Registriert seit | nein | Datum der Aktivierung |

### Einspeisungsart (`generation_type`)

Das Feld `generation_type` klassifiziert den Erzeugungstyp und wird in EDA-Nachrichten sowie in Berichten ausgewiesen. Es steht seit Datenbankmigration 029 zur Verfügung.

### Hinweis zum Verteilungsmodell

Aktuell ist in `eegabrechnung` nur das dynamische Aufteilungsmodell implementiert. Eine statische Aufteilung ist derzeit nicht produktiv umgesetzt, auch wenn einzelne Felder oder UI-Optionen dafür bereits vorhanden sein können.

| Wert | Bedeutung |
|------|-----------|
| `PV` | Photovoltaik |
| `Windkraft` | Windkraftanlage |
| `Wasserkraft` | Wasserkraftanlage |
| `Biomasse` | Biomasseanlage |
| `Sonstige` | Sonstige Erzeugungsart |

### Zählpunkt-Notizen

Zu jedem Zählpunkt können interne **Notizen** hinterlegt werden — z.B. „Zähler wurde am 15.3. getauscht", „Netzbetreiber meldet Probleme", Kontaktperson beim NB. Die Notizen sind nur für den EEG-Administrator sichtbar und erscheinen auf der Mitglieds-Detailseite.

Über `PUT /api/v1/eegs/{eegId}/members/{memberId}/meter-points/{mpId}` wird das Feld `notes` mitgeschickt.

### Consent-ID (EDA-Zustimmungskennung)

Wenn ein Zählpunkt vom Netzbetreiber im EDA-System angemeldet wird (EC_REQ_ONL bestätigt), sendet der Netzbetreiber in der ECMPList-Bestätigung eine **ConsentId**. Diese wird automatisch auf dem Zählpunkt gespeichert (`consent_id`) und ist Voraussetzung für das spätere Widerruf-Verfahren (CM_REV_SP).

Auf der Mitglieds-Detailseite wird der EDA-Status jedes Zählpunkts angezeigt:
- `registriert_seit` — Datum der erfolgreichen Anmeldung
- `abgemeldet_am` — Datum der bestätigten Abmeldung (CM_REV_SP)
- Anmeldungs-/Abmeldungsstatus als Badge (pending/sent/confirmed/completed/rejected/error)

### EDA-Anmeldung beim Anlegen

Wenn die EDA-Kommunikation in den EEG-Einstellungen konfiguriert ist (Marktpartner-ID und Netzbetreiber-ID vorhanden), kann beim Anlegen eines Zählpunkts sofort eine EDA-Anmeldung (`EC_EINZEL_ANM`) ausgelöst werden. Hierfür sind anzugeben:

| Feld | Beschreibung |
|------|-------------|
| Gültig ab | Startdatum der Teilnahme |
| Teilnahmefaktor (%) | Anteil der zugeteilten Gemeinschaftsenergie (Standard: 100 %) |
| Anteilstyp | `GC` (Vollzuteilung), `RC_R`, `RC_L`, `CC` |

<div class="tip">
Die EDA-Anmeldung kann jederzeit auch nachträglich über die EDA-Seite (<code>/eegs/{eegId}/eda</code>) ausgelöst werden — eine sofortige Auslösung beim Anlegen des Zählpunkts ist optional.
</div>

<div class="warning">
Ohne korrekte EDA-Konfiguration in den EEG-Einstellungen ist die Checkbox für die sofortige Anmeldung deaktiviert. EDA-IDs zuerst unter <em>Einstellungen → Tab EDA</em> eintragen.
</div>

---

## Mitglied abmelden (Austritt)

Der Austritt eines Mitglieds löst automatisch den Widerruf für alle aktiven Zählpunkte aus.

**Ablauf:**
1. Mitglieds-Detailseite öffnen
2. Button „Mitglied abmelden" klicken (nur bei ACTIVE-Mitgliedern sichtbar)
3. Austrittsdatum wählen (frühestens morgen, Arbeitstage-Beschränkung beachten)
4. Bestätigen

**Was passiert im Hintergrund:**
- Mitglied wird auf Status `INACTIVE` gesetzt, `austritt_datum` wird gespeichert
- Für jeden Zählpunkt mit gespeicherter `consent_id`: CM_REV_SP (Widerruf) wird an den Netzbetreiber gesendet mit `valid_from = austritt_datum`
- Zählpunkte ohne `consent_id` werden übersprungen (nie offiziell angemeldet)
- Idempotent: wenn für einen Zählpunkt bereits ein Widerruf läuft (Status pending/sent), wird er übersprungen

**Endpunkt:** `POST /api/v1/eegs/{eegId}/members/{memberId}/austritt`
**Body:** `{ "austritt_datum": "YYYY-MM-DD" }`

<div class="warning">Das Austrittsdatum muss mindestens morgen sein und kann maximal 30 österreichische Arbeitstage in der Zukunft liegen (SEPA-Rulebook und EDA-Anforderungen). Das Abschlussrechnung-Erstellen ist ein separater Schritt — der Austritt erstellt keine Rechnung automatisch.</div>

---

## SEPA-Lastschrift-Mandat

Für Mitglieder, die über das Onboarding-Portal beigetreten sind und dabei das SEPA-Mandat unterzeichnet haben, kann das Mandat als PDF heruntergeladen werden.

**Endpunkt:** `GET /api/v1/eegs/{eegId}/members/{memberId}/sepa-mandat`

Das PDF enthält:
- Mandatsreferenz (Mitglieds-UUID)
- Name und Adresse des Zahlungspflichtigen
- IBAN der EEG
- Unterzeichnungsdatum und IP-Adresse
- Volltext des akzeptierten Mandatstexts

<div class="tip">Das Mandat wird automatisch beim Onboarding erfasst wenn der Bewerber im Registrierungsformular die SEPA-Lastschrift-Berechtigung erteilt. Die Metadaten (Zeitstempel, IP, Text) werden gemäß SEPA-Rulebook revisionssicher gespeichert.</div>

---

## 4.4 Mehrfachteilnahme

Ein Zählpunkt kann seit dem EAG April 2024 an mehreren Energiegemeinschaften gleichzeitig teilnehmen. Die Verwaltung der Teilnahmefaktoren erfolgt über die separate Seite `/eegs/{eegId}/participations` (Kapitel 14).
