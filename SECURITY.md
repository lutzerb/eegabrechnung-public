# Security

## Sicherheitsluecken melden

Bitte melde Sicherheitsluecken **nicht** als oeffentliches GitHub-Issue.

Stattdessen bitte direkt per E-Mail an den Maintainer (Kontakt ueber das GitHub-Profil). Alternativ kann auch ein privates GitHub Security Advisory ueber den "Report a vulnerability"-Link im Repo verwendet werden.

Ich bestatige den Eingang in der Regel innerhalb weniger Tage und informiere ueber den weiteren Ablauf.

## Selbsthosting — eigene Verantwortung

Wer `eegabrechnung` selbst hostet, traegt die Verantwortung fuer die Betriebssicherheit der eigenen Instanz. Dazu gehoeren insbesondere:

- TLS erzwingen (kein HTTP im Produktivbetrieb)
- Starke, zufaellig generierte Secrets fuer `JWT_SECRET`, `NEXTAUTH_SECRET` und `CREDENTIAL_ENCRYPTION_KEY`
- Postgres nicht direkt ins Internet exponieren
- API-Port 8101 nicht direkt ins Internet exponieren (nur Port 3001 hinter Reverse-Proxy)
- Regelmaessige Datenbankbackups mit getesteter Wiederherstellung
- `.env`-Datei mit Secrets aus dem Repo ausschliessen (`.gitignore` beachten)

## Bekannte Einschraenkungen

- Die Software ist auf den oesterreichischen EEG-Kontext ausgelegt und wurde nicht gegen ein formales Sicherheits-Audit eingereicht.
- EDA-Credentials (IMAP/SMTP) werden AES-256-GCM-verschluesselt in der Datenbank gespeichert. Der Verschluesselungsschluessel (`CREDENTIAL_ENCRYPTION_KEY`) muss sicher verwahrt werden.
