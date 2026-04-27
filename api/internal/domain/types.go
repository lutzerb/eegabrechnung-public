package domain

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant/customer.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// User represents a login user within an organization.
type User struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	Name           string    `json:"name"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

// EDAMessage represents an EDA protocol message.
type EDAMessage struct {
	ID          uuid.UUID  `json:"id"`
	MessageID   string     `json:"message_id"`   // external message-id from MaKo header
	Direction   string     `json:"direction"`
	Process     string     `json:"process"`      // EDA process code, e.g. EC_EINZEL_ANM, DATEN_CRMSG
	MessageType string     `json:"message_type"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body,omitempty"`         // plain-text email body (MAIL transport)
	FromAddress string     `json:"from_address,omitempty"` // sender EDA address
	ToAddress   string     `json:"to_address,omitempty"`   // recipient EDA address
	Status      string     `json:"status"`                 // pending | sent | ack | error | processed
	ErrorMsg    string     `json:"error_msg,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// EEGStats holds aggregate statistics for an EEG.
type EEGStats struct {
	MemberCount     int        `json:"member_count"`
	MeterPointCount int        `json:"meter_point_count"`
	InvoiceCount    int        `json:"invoice_count"`
	BillingRunCount int        `json:"billing_run_count"`
	TotalKwh        float64    `json:"total_kwh"`         // sum of consumption_kwh from invoices
	TotalRevenue    float64    `json:"total_revenue"`     // sum of positive total_amount (consumer charges)
	LastBillingRun  *time.Time `json:"last_billing_run,omitempty"`
}

// EEG represents an Energiegemeinschaft.
type EEG struct {
	ID                  uuid.UUID `json:"id"`
	OrganizationID      uuid.UUID `json:"organization_id"`
	GemeinschaftID      string    `json:"gemeinschaft_id"`
	Netzbetreiber       string    `json:"netzbetreiber"`
	Name                string    `json:"name"`
	EnergyPrice         float64   `json:"energy_price"`          // consumer work price ct/kWh (net)
	ProducerPrice       float64   `json:"producer_price"`        // producer feed-in price ct/kWh (net)
	UseVat              bool      `json:"use_vat"`               // include VAT on invoices
	VatPct              float64   `json:"vat_pct"`               // VAT percentage (e.g. 20)
	MeterFeeEur         float64   `json:"meter_fee_eur"`         // fixed fee per consumer meter point per period
	FreeKwh             float64   `json:"free_kwh"`              // complimentary kWh per member per period
	DiscountPct         float64   `json:"discount_pct"`          // discount on consumption in %
	ParticipationFeeEur float64   `json:"participation_fee_eur"` // fixed member participation fee per period
	BillingPeriod       string    `json:"billing_period"`        // monthly | quarterly | semiannual | annual
	InvoiceNumberPrefix string    `json:"invoice_number_prefix"`
	InvoiceNumberDigits int       `json:"invoice_number_digits"`
	InvoiceNumberStart  int       `json:"invoice_number_start"` // first invoice number for this EEG
	InvoicePreText      string    `json:"invoice_pre_text"`
	InvoicePostText     string    `json:"invoice_post_text"`
	InvoiceFooterText   string    `json:"invoice_footer_text"`
	// Company logo path (stored as absolute filesystem path)
	LogoPath string `json:"logo_path"`
	// Credit note settings (for VAT-liable producers)
	GenerateCreditNotes      bool   `json:"generate_credit_notes"`       // if true, pure producers with UID number get Gutschrift instead of Rechnung
	CreditNoteNumberPrefix   string `json:"credit_note_number_prefix"`   // e.g. "GS"
	CreditNoteNumberDigits   int    `json:"credit_note_number_digits"`   // e.g. 5
	// SEPA banking details for payment file generation
	IBAN           string `json:"iban"`             // EEG's own bank account IBAN
	BIC            string `json:"bic"`              // EEG's bank BIC (optional)
	SepaCreditorID string `json:"sepa_creditor_id"` // SEPA Gläubiger-ID for pain.008
	// EDA communication settings (Austrian MaKo protocol)
	EdaTransitionDate  *time.Time `json:"eda_transition_date,omitempty"` // date from which EDA email replaces XLSX import
	EdaMarktpartnerID  string     `json:"eda_marktpartner_id"`           // own EC/GC/RC number for outbound messages
	EdaNetzbetreiberID string     `json:"eda_netzbetreiber_id"`          // Netzbetreiber ECNumber to send messages to
	// Accounting / DATEV export configuration
	AccountingRevenueAccount int    `json:"accounting_revenue_account"` // Erlöskonto (default 4000)
	AccountingExpenseAccount int    `json:"accounting_expense_account"` // Aufwandskonto Einspeisung (default 5000)
	AccountingDebitorPrefix  int    `json:"accounting_debitor_prefix"`  // base for member debitor accounts (default 10000)
	DatevConsultantNr        string `json:"datev_consultant_nr"`        // DATEV Beraternummer (optional)
	DatevClientNr            string `json:"datev_client_nr"`            // DATEV Mandantennummer (optional)
	// Rechnungssteller (§11 UStG) — EEG address shown on invoices
	Strasse    string `json:"strasse"`
	Plz        string `json:"plz"`
	Ort        string `json:"ort"`
	UidNummer  string `json:"uid_nummer"` // EEG VAT ID (optional, not all EEGs are VAT-registered)
	// Founding date — coverage tool will not flag days before this as missing
	Gruendungsdatum *time.Time `json:"gruendungsdatum,omitempty"`
	// Onboarding contract text shown to applicants (may include {iban} and {datum} placeholders)
	OnboardingContractText string `json:"onboarding_contract_text"`
	// Per-EEG EDA credentials (IMAP polling + SMTP send to edanet.at).
	// Passwords are stored encrypted in the DB; the repository decrypts them.
	// Returned as "***" via API if set; send empty or "***" to keep existing.
	EDAIMAPHost     string `json:"eda_imap_host"`
	EDAIMAPUser     string `json:"eda_imap_user"`
	EDAIMAPPassword string `json:"eda_imap_password,omitempty"`

	EDASmtpHost     string `json:"eda_smtp_host"`
	EDASmtpUser     string `json:"eda_smtp_user"`
	EDASmtpPassword string `json:"eda_smtp_password,omitempty"`
	EDASmtpFrom     string `json:"eda_smtp_from"`

	// Per-EEG invoice SMTP credentials (Rechnungsversand via resend / own SMTP).
	SMTPHost     string `json:"smtp_host"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	SMTPFrom     string `json:"smtp_from"`

	// SepaPreNotificationDays is the minimum days between invoice date and SEPA collection.
	// SEPA Rulebook mandates ≥14 days. Configurable per EEG. Default: 14.
	SepaPreNotificationDays int `json:"sepa_pre_notification_days"`
	// IsDemo marks this EEG as a demo — emails and EDA messages are suppressed
	IsDemo    bool      `json:"is_demo"`
	// Auto-billing: create draft billing run automatically on a fixed day each month/quarter
	AutoBillingEnabled    bool       `json:"auto_billing_enabled"`
	AutoBillingDayOfMonth int        `json:"auto_billing_day_of_month"` // 1–28
	AutoBillingPeriod     string     `json:"auto_billing_period"`       // "monthly"|"quarterly"
	AutoBillingLastRunAt  *time.Time `json:"auto_billing_last_run_at,omitempty"`
	// Gap alert: notify when meter points have no readings for N days
	GapAlertEnabled       bool `json:"gap_alert_enabled"`
	GapAlertThresholdDays int  `json:"gap_alert_threshold_days"` // default 5
	// Portal: whether to show full energy data (total consumption/generation) in the member portal
	PortalShowFullEnergy bool `json:"portal_show_full_energy"`
	CreatedAt            time.Time `json:"created_at"`
}

// Member represents a member of an EEG.
//
// VAT rules (per Austrian EEG law):
//
//   - Consumer invoices (Rechnung): VAT is determined solely by the EEG's
//     UseVat/VatPct settings. Member-level VAT fields are NOT applied here.
//
//   - Producer credit notes (Gutschrift, not yet implemented): VAT is
//     determined by the member's properties:
//     - UidNummer present           → Reverse Charge (0% + RC text)
//     - BusinessRole == "landwirt_pauschaliert" → 13% + § 22 UStG text
//     - BusinessRole == "gemeinde_hoheitlich"   → 0% + sovereignty text
//     - otherwise (Privatperson / Kleinunternehmer) → 0% + § 6 exemption text
//
//   - UseVat / VatPct on the member are reserved for future manual overrides
//     on credit notes and are not applied to consumer invoices.
type Member struct {
	ID           uuid.UUID `json:"id"`
	EegID        uuid.UUID `json:"eeg_id"`
	MitgliedsNr  string    `json:"mitglieds_nr"`
	Name1        string    `json:"name1"`
	Name2        string    `json:"name2"`
	Email        string    `json:"email"`
	IBAN         string    `json:"iban"`
	Strasse      string    `json:"strasse"`
	Plz          string    `json:"plz"`
	Ort          string    `json:"ort"`
	// BusinessRole classifies the member for VAT purposes on credit notes.
	// Valid values: "privat", "kleinunternehmer", "verein", "landwirt_pauschaliert",
	// "unternehmen", "gemeinde_bga", "gemeinde_hoheitlich"
	BusinessRole string `json:"business_role"`
	// UidNummer is the member's VAT ID (UID).
	// When set, the member is treated as VAT-liable → Reverse Charge on credit notes.
	UidNummer string `json:"uid_nummer"`
	// UseVat / VatPct: reserved for future manual override on credit notes.
	// NOT applied to consumer invoices (those use only the EEG's settings).
	UseVat    *bool    `json:"use_vat"`
	VatPct    *float64 `json:"vat_pct"`
	// Status tracks the member's lifecycle: ACTIVE | REGISTERED | NEW | INACTIVE
	Status         string     `json:"status"`
	BeitrittsDatum *time.Time `json:"beitritts_datum,omitempty"`
	AustrittsDatum *time.Time `json:"austritts_datum,omitempty"`
	// SEPA mandate data captured during onboarding (used for mandate PDF)
	SepaMandateSignedAt *time.Time `json:"sepa_mandate_signed_at,omitempty"`
	SepaMandateSignedIP string     `json:"sepa_mandate_signed_ip,omitempty"`
	SepaMandateText     string     `json:"sepa_mandate_text,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// MeterPoint represents a Zählpunkt.
type MeterPoint struct {
	ID                  uuid.UUID  `json:"id"`
	MemberID            uuid.UUID  `json:"member_id"`
	EegID               uuid.UUID  `json:"eeg_id"`
	Zaehlpunkt          string     `json:"zaehlpunkt"`
	Energierichtung     string     `json:"energierichtung"`
	Verteilungsmodell   string     `json:"verteilungsmodell"`
	ZugeteilteMenugePct float64    `json:"zugeteilte_menge_pct"`
	Status              string     `json:"status"`
	RegistriertSeit     *time.Time `json:"registriert_seit,omitempty"`
	AbgemeldetAm        *time.Time `json:"abgemeldet_am,omitempty"`
	GenerationType      *string    `json:"generation_type,omitempty"` // PV | Windkraft | Wasserkraft; only for GENERATION meters
	GapAlertSentAt      *time.Time `json:"gap_alert_sent_at,omitempty"`
	Notes               string     `json:"notes"`
	ConsentID           string     `json:"consent_id"` // NB-assigned ConsentId from ZUSTIMMUNG_ECON; required for CM_REV_SP
	CreatedAt           time.Time  `json:"created_at"`
}

// EnergyReading represents a single 15-minute energy reading.
type EnergyReading struct {
	ID           uuid.UUID `json:"id"`
	MeterPointID uuid.UUID `json:"meter_point_id"`
	Ts           time.Time `json:"ts"`
	WhTotal      float64   `json:"wh_total"`
	WhCommunity  float64   `json:"wh_community"`
	WhSelf       float64   `json:"wh_self"`
	Source       string    `json:"source"`  // "xlsx" or "eda"
	Quality      string    `json:"quality"` // L0 (total), L1 (measured), L2 (substitute), L3 (faulty — excluded from billing)
	CreatedAt    time.Time `json:"created_at"`
}

// EDAError represents a failed inbound EDA message stored for operator review.
type EDAError struct {
	ID          uuid.UUID  `json:"id"`
	EegID       *uuid.UUID `json:"eeg_id,omitempty"`
	Direction   string     `json:"direction"`
	MessageType string     `json:"message_type"`
	Subject     string     `json:"subject"`
	RawContent  string     `json:"raw_content"`
	ErrorMsg    string     `json:"error_msg"`
	CreatedAt   time.Time  `json:"created_at"`
}

// EDAWorkerStatus holds the last-known state of the EDA worker.
type EDAWorkerStatus struct {
	TransportMode string     `json:"transport_mode"`
	LastPollAt    *time.Time `json:"last_poll_at,omitempty"`
	LastError     string     `json:"last_error"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// BillingRun groups all invoices created in one billing operation.
type BillingRun struct {
	ID           uuid.UUID `json:"id"`
	EegID        uuid.UUID `json:"eeg_id"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	Status       string    `json:"status"`
	InvoiceCount int       `json:"invoice_count"`
	TotalAmount  float64   `json:"total_amount"`
	CreatedAt    time.Time `json:"created_at"`
}

// EEGMeterParticipation records a meter point's participation in an EEG
// with a given factor and date range (Mehrfachteilnahme, Austrian EAG).
type EEGMeterParticipation struct {
	ID                  uuid.UUID  `json:"id"`
	EegID               uuid.UUID  `json:"eeg_id"`
	MeterPointID        uuid.UUID  `json:"meter_point_id"`
	ParticipationFactor float64    `json:"participation_factor"` // 0.0001–100
	ShareType           string     `json:"share_type"`           // GC | RC_R | RC_L | CC
	ValidFrom           time.Time  `json:"valid_from"`
	ValidUntil          *time.Time `json:"valid_until,omitempty"` // nil = open-ended
	Notes               string     `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
}

// Invoice represents a billing invoice for a member.
type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	MemberID       uuid.UUID  `json:"member_id"`
	EegID          uuid.UUID  `json:"eeg_id"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	TotalKwh       float64    `json:"total_kwh"`        // consumption kWh (for display/stats)
	TotalAmount    float64    `json:"total_amount"`     // gross saldo incl. VAT (may be negative for producers)
	NetAmount      float64    `json:"net_amount"`       // net amount before VAT
	VatAmount      float64    `json:"vat_amount"`       // total VAT (consumption + generation)
	VatPctApplied  float64    `json:"vat_pct_applied"`  // EEG-level VAT rate (consumption side)
	ConsumptionKwh float64    `json:"consumption_kwh"` // community kWh consumed
	GenerationKwh  float64    `json:"generation_kwh"`  // community kWh generated/fed in
	// Split net amounts — stored separately to enable per-account EA import bookings
	ConsumptionNetAmount float64 `json:"consumption_net_amount"` // net Bezug (consumption charge before VAT)
	GenerationNetAmount  float64 `json:"generation_net_amount"`  // net Einspeisung (generation credit before VAT)
	// Split VAT — consumption uses EEG-level rate, generation uses member-specific rate
	ConsumptionVatPct    float64 `json:"consumption_vat_pct"`
	ConsumptionVatAmount float64 `json:"consumption_vat_amount"`
	GenerationVatPct     float64 `json:"generation_vat_pct"`
	GenerationVatAmount  float64 `json:"generation_vat_amount"`
	PdfPath        string     `json:"pdf_path"`
	StornoPdfPath  string     `json:"storno_pdf_path"` // set when invoice is formally cancelled (storno document)
	SentAt         *time.Time `json:"sent_at,omitempty"`
	InvoiceNumber  *int       `json:"invoice_number,omitempty"`
	Status         string     `json:"status"`
	DocumentType   string     `json:"document_type"` // "invoice" or "credit_note"
	BillingRunID   *uuid.UUID `json:"billing_run_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	// SEPA return (Rücklastschrift) tracking
	SepaReturnAt     *time.Time `json:"sepa_return_at,omitempty"`
	SepaReturnReason string     `json:"sepa_return_reason,omitempty"`
	SepaReturnNote   string     `json:"sepa_return_note,omitempty"`
}

// MonthlyEnergyRow holds aggregated energy and financial data for one calendar month.
type MonthlyEnergyRow struct {
	Month          time.Time `json:"month"`
	ConsumptionKwh float64   `json:"consumption_kwh"`
	GenerationKwh  float64   `json:"generation_kwh"`
	Revenue        float64   `json:"revenue"` // sum of positive invoice totals (consumer charges incl. VAT)
	Payouts        float64   `json:"payouts"` // absolute sum of negative invoice totals (producer credits)
}

// MemberStat holds aggregated invoice totals for one member over a time range.
type MemberStat struct {
	MemberID            uuid.UUID `json:"member_id"`
	ConsumptionKwh      float64   `json:"consumption_kwh"`       // EEG share consumed (wh_self)
	GenerationKwh       float64   `json:"generation_kwh"`        // EEG share fed in (wh_community)
	ConsumptionTotalKwh float64   `json:"consumption_total_kwh"` // total consumption (wh_total); 0 for billed mode
	GenerationTotalKwh  float64   `json:"generation_total_kwh"`  // total generation (wh_total); 0 for billed mode
	TotalAmount         float64   `json:"total_amount"`
	InvoiceCount        int       `json:"invoice_count"`
}

// AnnualReportMeterPoint is one meter point entry in the annual report member list.
type AnnualReportMeterPoint struct {
	Zaehlpunkt      string     `json:"zaehlpunkt"`
	Energierichtung string     `json:"energierichtung"`
	GenerationType  *string    `json:"generation_type,omitempty"`
	RegistriertSeit *time.Time `json:"registriert_seit,omitempty"`
	AbgemeldetAm    *time.Time `json:"abgemeldet_am,omitempty"`
}

// AnnualReportMember is one member row in the annual report.
type AnnualReportMember struct {
	MemberID       uuid.UUID                `json:"member_id"`
	MitgliedsNr    string                   `json:"mitglieds_nr"`
	Name           string                   `json:"name"`
	Email          string                   `json:"email"`
	Strasse        string                   `json:"strasse"`
	Plz            string                   `json:"plz"`
	Ort            string                   `json:"ort"`
	MemberType     string                   `json:"member_type"`
	BeitrittsDatum *time.Time               `json:"beitritts_datum,omitempty"`
	AustrittsDatum *time.Time               `json:"austritts_datum,omitempty"`
	Zaehlpunkte    []AnnualReportMeterPoint `json:"zaehlpunkte"`
	WhConsumption  float64                  `json:"wh_consumption"`
	WhGeneration   float64                  `json:"wh_generation"`
	WhCommunity    float64                  `json:"wh_community"`
	Invoiced       float64                  `json:"invoiced"`
	Credited       float64                  `json:"credited"`
	InvoiceCount   int                      `json:"invoice_count"`
}

// EnergySummaryRow holds raw energy metrics aggregated from energy_readings for a time bucket.
type EnergySummaryRow struct {
	Period              time.Time `json:"period"`
	WhSelf              float64   `json:"wh_self"`               // EEG share consumed (CONSUMPTION wh_self)
	WhCommunity         float64   `json:"wh_community"`          // EEG share fed in (GENERATION wh_community)
	WhTotalConsumption  float64   `json:"wh_total_consumption"`  // total consumption (CONSUMPTION wh_total)
	WhTotalGeneration   float64   `json:"wh_total_generation"`   // total generation (GENERATION wh_total)
	WhRestbedarf        float64   `json:"wh_restbedarf"`         // grid demand = wh_total_consumption - wh_self
	WhResteinspeisung   float64   `json:"wh_resteinspeisung"`    // grid export  = wh_total_generation - wh_community
}

// EDAProcess tracks an open EDA protocol process (Anmeldung, Abmeldung, Teilnahmefaktor).
type EDAProcess struct {
	ID                  uuid.UUID  `json:"id"`
	EegID               uuid.UUID  `json:"eeg_id"`
	MeterPointID        *uuid.UUID `json:"meter_point_id,omitempty"`
	ProcessType         string     `json:"process_type"`    // EC_REQ_ONL, EC_REQ_OFF, CM_REV_SP, EC_PRTFACT_CHG, EC_REQ_PT
	Status              string     `json:"status"`          // pending, sent, first_confirmed, confirmed, completed, rejected, error
	ConversationID      string     `json:"conversation_id"` // links outbound to inbound confirmations
	Zaehlpunkt          string     `json:"zaehlpunkt"`
	ValidFrom           *time.Time `json:"valid_from,omitempty"`
	ParticipationFactor *float64   `json:"participation_factor,omitempty"`
	ShareType           string     `json:"share_type"` // GC, RC_R, RC_L, CC, NONE, MULTI
	// ECMPList-specific fields (EC_PRTFACT_CHG)
	ECDisModel      string     `json:"ec_dis_model"`           // S = statisch, D = dynamisch
	DateTo          *time.Time `json:"date_to,omitempty"`      // ECMPList DateTo
	EnergyDirection string     `json:"energy_direction"`       // CONSUMPTION or GENERATION
	ECShare         *float64   `json:"ec_share,omitempty"`     // optional share % in static model
	InitiatedAt                 time.Time  `json:"initiated_at"`
	DeadlineAt                  *time.Time `json:"deadline_at,omitempty"` // 2 months for ANM (EAG §16e)
	CompletedAt                 *time.Time `json:"completed_at,omitempty"`
	ErrorMsg                    string     `json:"error_msg"`
	ErrorNotificationSentAt     *time.Time `json:"error_notification_sent_at,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	// Joined field — only populated by ListByEEG, empty in other queries.
	MemberName string `json:"member_name,omitempty"`
}

// Job represents an async job.
type Job struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Payload   []byte    `json:"payload"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OnboardingMeterPoint is a meter point entry in an onboarding request.
type OnboardingMeterPoint struct {
	Zaehlpunkt          string  `json:"zaehlpunkt"`
	Direction           string  `json:"direction"`                     // CONSUMPTION or GENERATION
	GenerationType      string  `json:"generation_type,omitempty"`     // PV | Windkraft | Wasserkraft | Biomasse | Sonstige — only for GENERATION
	ParticipationFactor float64 `json:"participation_factor,omitempty"` // 0..100; 0 means default (100)
}

// OnboardingRequest tracks a new member signing up for an EEG.
// Status flow: pending → converted (approve triggers member creation) or rejected.
type OnboardingRequest struct {
	ID                  uuid.UUID              `json:"id"`
	EegID               uuid.UUID              `json:"eeg_id"`
	Status              string                 `json:"status"`
	Name1               string                 `json:"name1"`
	Name2               string                 `json:"name2"`
	Email               string                 `json:"email"`
	Phone               string                 `json:"phone"`
	Strasse             string                 `json:"strasse"`
	PLZ                 string                 `json:"plz"`
	Ort                 string                 `json:"ort"`
	IBAN                string                 `json:"iban"`
	BIC                 string                 `json:"bic"`
	MemberType          string                 `json:"member_type"`   // CONSUMER | PRODUCER | PROSUMER
	BusinessRole        string                 `json:"business_role"` // privat | kleinunternehmer | ...
	UidNummer           string                 `json:"uid_nummer"`
	UseVat              bool                   `json:"use_vat"`
	MeterPoints         []OnboardingMeterPoint `json:"meter_points"`
	BeitrittsDatum      *time.Time             `json:"beitritts_datum,omitempty"` // desired join date from applicant
	ContractAcceptedAt  *time.Time             `json:"contract_accepted_at,omitempty"`
	ContractIP          string                 `json:"contract_ip"`
	MagicToken          string                 `json:"-"`
	MagicTokenExpiresAt time.Time              `json:"-"`
	AdminNotes          string                 `json:"admin_notes"`
	ConvertedMemberID   *uuid.UUID             `json:"converted_member_id,omitempty"`
	ReminderSentAt      *time.Time             `json:"reminder_sent_at,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// StammdatenRow holds parsed data from a single Stammdaten XLSX row.
type StammdatenRow struct {
	Netzbetreiber       string
	GemeinschaftID      string
	Zaehlpunkt          string
	Energierichtung     string
	Verteilungsmodell   string
	ZugeteilteMenugePct float64
	Name1               string
	Name2               string
	Email               string
	IBAN                string
	BusinessRole        string
	MitgliedsNr         string
	Zaehlpunktstatus    string
	RegistriertSeit     string
}

// EnergyRow holds a single parsed row from an energy XLSX.
type EnergyRow struct {
	MeterID     string
	Ts          time.Time
	WhTotal     float64
	WhCommunity float64
	WhSelf      float64
}

// CampaignAttachment holds metadata for an email attachment.
type CampaignAttachment struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	FilePath string `json:"-"`
}

// MemberEmailCampaign records a bulk email sent to all EEG members.
type MemberEmailCampaign struct {
	ID             uuid.UUID            `json:"id"`
	EegID          uuid.UUID            `json:"eeg_id"`
	Subject        string               `json:"subject"`
	HtmlBody       string               `json:"html_body"`
	RecipientCount int                  `json:"recipient_count"`
	Attachments    []CampaignAttachment `json:"attachments"`
	CreatedAt      time.Time            `json:"created_at"`
}

// EEGDocument is a file downloadable by members in the portal.
type EEGDocument struct {
	ID               uuid.UUID `json:"id"`
	EegID            uuid.UUID `json:"eeg_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Filename         string    `json:"filename"`
	FilePath         string    `json:"-"` // server-side path, not exposed
	MimeType         string    `json:"mime_type"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	SortOrder        int       `json:"sort_order"`
	ShowInOnboarding bool      `json:"show_in_onboarding"`
	CreatedAt        time.Time `json:"created_at"`
}

// TariffSchedule is a named pricing model with time-varying entries.
type TariffSchedule struct {
	ID          uuid.UUID     `json:"id"`
	EegID       uuid.UUID     `json:"eeg_id"`
	Name        string        `json:"name"`
	Granularity string        `json:"granularity"` // annual | monthly | daily | quarter_hour
	IsActive    bool          `json:"is_active"`
	EntryCount  int           `json:"entry_count,omitempty"`
	Entries     []TariffEntry `json:"entries,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

// TariffEntry is a price valid within a time range.
type TariffEntry struct {
	ID            uuid.UUID `json:"id"`
	ScheduleID    uuid.UUID `json:"schedule_id"`
	ValidFrom     time.Time `json:"valid_from"`
	ValidUntil    time.Time `json:"valid_until"`
	EnergyPrice   float64   `json:"energy_price"`   // ct/kWh
	ProducerPrice float64   `json:"producer_price"` // ct/kWh
	CreatedAt     time.Time `json:"created_at"`
}
