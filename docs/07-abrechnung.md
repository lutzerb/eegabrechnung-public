# Abrechnungsläufe & Rechnungen

![Abrechnungsläufe Übersicht](screenshots/billing.png)

---

## Konzept

Ein **Abrechnungslauf** fasst alle Rechnungen einer einzelnen Abrechnungsoperation zusammen. Das System erkennt Zeitraum-Überschneidungen und verhindert doppelte Abrechnungen automatisch (HTTP 409 bei Konflikt).

Jeder Abrechnungslauf durchläuft folgende Zustände:

```
[Neu anlegen] → draft ──── [Finalisieren] ──→ finalized
                                            ├─ [Stornieren] → cancelled
                                            └─ [SEPA-Export]
```

---

## Abrechnungslauf erstellen

Über **Neuer Abrechnungslauf** öffnet sich ein Formular:

| Feld | Beschreibung |
|------|-------------|
| **Von** | Startdatum des Abrechnungszeitraums |
| **Bis** | Enddatum des Abrechnungszeitraums |
| **Mitgliederfilter** | Alle / nur Consumer / nur Producer |
| **Typ** | Rechnung (`invoice`) / Gutschrift (`credit_note`) |
| **Force-Override** | Überschreibt die Überlappungserkennung (Vorsicht!) |

<div class="warning">
Die Überlappungserkennung verhindert, dass derselbe Zeitraum doppelt abgerechnet wird. Die Option **Force-Override** deaktiviert diese Prüfung und sollte nur in begründeten Ausnahmefällen verwendet werden (z. B. Nachkorrektur bei Datenfehler).
</div>

Nach der Erstellung befinden sich alle generierten Rechnungen im Status **Entwurf** und können noch geprüft werden.

---

## Workflow & Status

### Abrechnungslauf-Status

| Status | Bedeutung |
|--------|-----------|
| `draft` | Entwurf — noch nicht finalisiert, kann gelöscht werden |
| `finalized` | Finalisiert — unveränderbar, SEPA-Export verfügbar |
| `cancelled` | Storniert — Storno-PDFs wurden erzeugt |

### Rechnungs-Status

| Status | Bedeutung |
|--------|-----------|
| **Entwurf** | Noch nicht finalisiert, kann gelöscht werden |
| **Finalisiert** | Abgerechnet, unveränderlich |
| **Versendet** | Per E-Mail verschickt |
| **Bezahlt** | Manuell als bezahlt markiert |
| **Storniert** | Stornierte Rechnung (bleibt sichtbar) |

---

## Rechnungsfelder

Eine Rechnung enthält folgende zentrale Datenfelder:

| Feld | Beschreibung | Seit |
|------|-------------|------|
| `member_id` | Zugehöriges Mitglied | 001 |
| `billing_run_id` | Zugehöriger Abrechnungslauf | 011 |
| `period_from` / `period_to` | Abrechnungszeitraum | 001 |
| `consumption_kwh` | Verbrauchte Energie (kWh) | 009 |
| `generation_kwh` | Eingespeiste Energie (kWh) | 009 |
| `net_amount` | Nettobetrag (EUR) | 025 |
| `vat_amount` | Umsatzsteuerbetrag (EUR) | 025 |
| `vat_pct_applied` | Angewandter USt.-Satz (%) | 025 |
| `consumption_net_amount` | Nettobetrag des Verbrauchsanteils (EUR) | 063 |
| `generation_net_amount` | Nettobetrag des Erzeugungsanteils (EUR) | 063 |
| `invoice_number` | Rechnungsnummer (Präfix + Nummer) | 004 |
| `document_type` | `invoice` oder `credit_note` | 019 |
| `status` | Aktueller Status der Rechnung | 004 |
| `storno_pdf_path` | Pfad zum Storno-PDF | 026 |

<div class="tip">
Rechnungsnummer-Präfix und Stellenanzahl werden in den EEG-Einstellungen konfiguriert (`invoice_number_prefix`, `invoice_number_digits`). Ebenso gibt es separate Einstellungen für Gutschriften (`credit_note_number_prefix`, `credit_note_number_digits`).
</div>

---

## Aufschlüsselung nach Verbrauch und Erzeugung

Seit Migration 063 enthält jede Rechnung eine Aufschlüsselung des Nettobetrags nach Verbrauch und Erzeugung:

| DB-Spalte | Bedeutung |
|-----------|----------|
| `consumption_net_amount` | Nettobetrag des Verbrauchsanteils |
| `generation_net_amount` | Nettobetrag des Erzeugungsanteils |

Diese Werte sind relevant für die steuerliche Differenzierung (z.B. 10% USt auf Lieferung aus erneuerbaren Quellen vs. 20% auf Netznutzung) und für die E/A-Buchhaltung.

Für **Prosumer** (Mitglieder die gleichzeitig einspeisen und beziehen) werden diese Beträge exakt berechnet wenn die Rechnung USt-pflichtig ist (`use_vat = true`). Für Kleinunternehmer (KU) erfolgt die Berechnung über den Billingservice.

---

## Gutschriften (credit_notes)

VAT-pflichtige Produzenten erhalten statt negativer Rechnungen **Gutschriften** (österreichische Praxis nach §11 UStG).

**Aktivierung:**

1. EEG-Einstellungen öffnen
2. Option **Gutschriften generieren** aktivieren (`generate_credit_notes = true`)
3. Optional: eigenen Nummernkreis konfigurieren (`credit_note_number_prefix` / `credit_note_number_digits`)

**Verhalten:**

- Produzenten mit `use_vat = true` erhalten automatisch `document_type = credit_note`
- Gutschriften verwenden denselben Abrechnungslauf wie reguläre Rechnungen
- Getrennte Nummerierung ermöglicht lückenlose Buchführung

---

## Finalisieren & Stornieren

### Finalisieren

Durch Finalisierung wird der Abrechnungslauf von `draft` auf `finalized` gesetzt:

- Alle zugehörigen Rechnungen werden auf `finalized` gesetzt
- PDFs werden generiert und gespeichert
- SEPA-Export wird freigeschaltet
- Änderungen sind danach nicht mehr möglich

<div class="danger">
Ein finalisierter Abrechnungslauf kann nicht mehr geändert werden. Fehlerhafte Rechnungen müssen storniert und neu erstellt werden.
</div>

### Stornieren

Bei Stornierung eines Abrechnungslaufs:

- Status wechselt zu `cancelled`
- Für jede Rechnung wird ein **Storno-PDF** generiert (`storno_pdf_path` in DB)
- Stornierte Rechnungen bleiben dauerhaft sichtbar (Prüfpfad / Revisionsschutz)
- Ein neuer, korrigierter Abrechnungslauf kann danach erstellt werden

### Storno-PDF

Wenn ein Abrechnungslauf storniert wird, wird für jede betroffene Rechnung automatisch ein **Storno-PDF** erstellt. Der Pfad wird in `storno_pdf_path` auf der Rechnung gespeichert und kann über die Rechnungsdetailseite heruntergeladen werden.

---

## SEPA-Export

Nach Finalisierung stehen zwei SEPA-Dateiformate zum Download bereit:

| Datei | Format | Verwendung |
|-------|--------|-----------|
| **pain.001** | SEPA Credit Transfer | Überweisungen an Produzenten (Gutschriften) |
| **pain.008** | SEPA Direct Debit | Lastschriften von Consumern (Rechnungen) |

Voraussetzung für den SEPA-Export: IBAN, BIC und SEPA-Gläubiger-ID müssen in den EEG-Einstellungen hinterlegt sein (`iban`, `bic`, `sepa_creditor_id`).

<div class="warning">
SEPA-Dateien werden direkt aus den finalisierten Rechnungsdaten generiert. Stellen Sie sicher, dass alle Mitglieder vollständige Bankverbindungsdaten hinterlegt haben, bevor Sie finalisieren.
</div>

---

## SEPA-Rücklastschriften

Wenn ein Lastschrifteinzug zurückgebucht wird, kann dies direkt auf der Rechnungsdetailseite erfasst werden.

### Manuelle Erfassung

Button „Rücklastschrift erfassen" auf der Rechnungsdetailseite:

| Feld | Beschreibung |
|------|-------------|
| Datum | Rücklastschrift-Datum |
| Grund | SEPA-Code: `AC01` (Falsche IBAN), `AM04` (kein Guthaben), `MS02` (abgelehnt durch Schuldner) etc. |
| Notiz | Interne Bemerkung |

Rechnungen mit Rücklastschrift erscheinen in der Liste mit einem **roten Badge**. Im EEG-Dashboard wird ein Alert angezeigt wenn offene Rücklastschriften vorliegen.

**Löschen:** Erneuter Klick auf „Rücklastschrift löschen" entfernt die Markierung.

### Automatischer Import (CAMT.054)

`POST /api/v1/eegs/{eegId}/sepa/camt054` — CAMT.054-XML importieren; matched Rücklastschriften automatisch per `EndToEndId = InvoiceUUID`.

→ Details in **[Kapitel 10: SEPA](10-sepa.md)**.

---

## Österreichische Umsatzsteuer-Matrix

Das System unterstützt die in Österreich üblichen USt.-Konstellationen für EEGs:

| Mitgliedstyp | Regelung | USt.-Satz |
|-------------|---------|----------|
| Consumer (Regelbesteuerung) | Standard | 20 % |
| Consumer (Kleinunternehmer) | §6 Abs. 1 Z 27 UStG | 0 % |
| Producer (Regelbesteuerung) | Gutschrift mit USt. | 20 % |
| Landwirt (Pauschalierung) | §22 UStG | 13 % |
| Reverse Charge (Bezug) | §19 Abs. 1 UStG | RC |

Per-Mitglied-Overrides sind über `use_vat` und `vat_pct` in der Mitgliederkonfiguration möglich (Migration 006).
