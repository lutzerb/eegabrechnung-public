# Buchhaltungsexport (DATEV)

Die Buchhaltungsseite ermöglicht den Export von Abrechnungsdaten in DATEV-kompatible Formate für die Übergabe an Steuerberater oder Buchhaltungssoftware.

**Seite:** `/eegs/{eegId}/accounting`

![Buchhaltungsexport](screenshots/accounting.png)

---

## Export-Formate

Der Export ist über einen frei wählbaren Zeitraum (von/bis) verfügbar und unterstützt zwei Formate:

| Format | Beschreibung |
|--------|-------------|
| `datev` | DATEV ASCII (EXTF-Format), direkt importierbar in DATEV Kanzlei-Rechnungswesen |
| `xlsx` | Excel-Tabelle mit denselben Feldern, für manuelle Weiterverarbeitung |

---

## Konfiguration der Konten

Die buchhalterischen Konten werden in den **EEG-Einstellungen → Tab System** hinterlegt.

| Feld | Bedeutung | Beispiel |
|------|-----------|---------|
| Erlöskonto | Umsatzerlöse Energielieferung (Verbraucher-Rechnungen) | 4400 |
| Aufwandskonto | Einkauf Einspeisung (Gutschriften an Produzenten) | 5400 |
| Debitoren-Konto | Forderungen aus Lieferungen und Leistungen | 1400 |
| Berater-Nummer | DATEV-Beraternummer des Steuerberaters | 12345 |
| Mandanten-Nummer | DATEV-Mandantennummer der EEG | 67890 |

<div class="tip">Werden keine Konten hinterlegt, bleibt der Export leer oder enthält Platzhalter. Die Konten müssen vor dem ersten Export vollständig konfiguriert sein.</div>

---

## MwSt-Aufschlüsselung

Die steuerlich relevanten Felder werden **zum Zeitpunkt der Abrechnung** unveränderlich gespeichert:

| DB-Spalte | Bedeutung |
|-----------|----------|
| `net_amount` | Nettobetrag der Rechnung/Gutschrift in EUR |
| `vat_amount` | MwSt-Betrag in EUR |
| `vat_pct_applied` | Angewendeter MwSt-Satz in Prozent |

Diese Werte werden **nicht** nachträglich aus den aktuellen Einstellungen neu berechnet — sie spiegeln exakt die steuerliche Situation zum Abrechnungszeitpunkt wider.

### Österreichische VAT-Matrix

| Konstellation | Regelung |
|--------------|---------|
| Standardmitglied | 20 % MwSt auf Energielieferung |
| Reverse Charge (RC) | 0 % / Steuerschuldumkehr beim Empfänger |
| Landwirt | Pauschalierungsregelung (§ 22 UStG) |
| Kleinunternehmer (KU) | Befreiung nach § 6 Abs. 1 Z 27 UStG |

Die pro-Mitglied-Konfiguration erfolgt in den Mitglieds-Stammdaten.

---

## DATEV-Felder im Export

Jede Buchungszeile enthält folgende Felder:

| Feld | Inhalt |
|------|--------|
| Buchungsdatum | Datum des Abrechnungslaufs |
| Belegdatum | Rechnungsdatum |
| Buchungstext | Mitgliedsname + Rechnungsnummer |
| Betrag | EUR-Betrag (positiv = Soll, negativ = Haben) |
| Konto | Erlös- oder Aufwandskonto (aus Einstellungen) |
| Gegenkonto | Debitoren-Konto (aus Einstellungen) |
| Steuercode | DATEV-kompatibler Steuerschlüssel entsprechend VAT-Matrix |

<div class="warning">Stornierte Rechnungen erzeugen eine Gegenbuchung. Stornos sind im Export als negative Beträge erkennbar. Vor dem Import in DATEV sicherstellen, dass stornierte und neue Rechnungen des gleichen Zeitraums gemeinsam exportiert werden.</div>

---

## Ablauf

1. Abrechnungslauf abschließen (Status: **finalized**)
2. Buchhaltungsseite öffnen (`/eegs/{eegId}/accounting`)
3. Zeitraum (von/bis) wählen — typischerweise der Abrechnungsmonat/-quartal
4. Format wählen (`datev` oder `xlsx`)
5. Exportdatei herunterladen und an Steuerberater übermitteln bzw. in DATEV importieren

---

## K2 Körperschaftsteuer-Erklärung (FinanzOnline XML)

Neben dem DATEV-Export steht für **Vereine mit wirtschaftlichem Geschäftsbetrieb** (EEG-Betrieb) ein direkter XML-Export für die FinanzOnline-Übermittlung zur Verfügung. Der K2-Export ist Teil des integrierten E/A-Buchhaltungsmoduls.

**Voraussetzung:** Das E/A-Buchhaltungsmodul muss eingerichtet sein (Steuernummer + Kontenplan mit K1-Kennzahl-Mapping, siehe **Kapitel 16**).

Der K2-Export aggregiert alle Buchungen des Jahres nach ihren K1-Kennzahlen und erzeugt eine BMF-konforme XML-Datei, die direkt in FinanzOnline hochgeladen werden kann.

**Ablauf:**
1. E/A-Buchhaltung → Jahresabschluss (`/eegs/{eegId}/ea/jahresabschluss`)
2. Jahr auswählen
3. „K2 XML exportieren" → Datei herunterladen
4. FinanzOnline → Erklärungen → K2 → Datei hochladen

<div class="tip">Der K2-Export ist für Vereine relevant, deren Energiebetrieb als wirtschaftlicher Geschäftsbetrieb körperschaftsteuerpflichtig ist. Für Genossenschaften oder andere Rechtsformen gelten abweichende Formulare — bitte mit dem Steuerberater klären.</div>

---

## Vollständige E/A-Buchhaltung (Kapitel 16)

Für Energiegemeinschaften mit Vereinsstatus steht ein integriertes **Einnahmen-Ausgaben-Rechnungs-Modul** zur Verfügung, das weit über den reinen DATEV-Export hinausgeht:

| Funktion | Beschreibung |
|----------|-------------|
| Kontenplan | Flexibler Kontenrahmen mit USt-Codes und FinanzOnline K1-Kennzahlen |
| Journal | Manuelle Buchungserfassung mit Belegen |
| Rechnungsimport | EEG-Abrechnungen automatisch als Buchungen übernehmen |
| Bankabgleich | MT940/CAMT.053 Import + Transaktionsmatching |
| Saldenliste / Kontenblatt | Auswertungen nach Konto und Zeitraum |
| Jahresabschluss (EAR) | Einnahmen-Ausgaben-Überschuss-Rechnung + K2 XML |
| UVA | Umsatzsteuer-Voranmeldung mit FinanzOnline XML-Export |
| Jahreserklärungen | U1 (Jahres-USt) + K1 (KSt-Basis §5 KStG) |
| BAO §131 Audit-Trail | Revisionssicherer Changelog aller Buchungsänderungen |

→ Vollständige Dokumentation in **[Kapitel 16: E/A-Buchhaltung](16-ea-buchhaltung.md)**.
