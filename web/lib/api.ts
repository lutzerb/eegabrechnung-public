// Typed API client for eegabrechnung backend

export interface EEG {
  id: string;
  gemeinschaft_id: string;
  netzbetreiber: string;
  name: string;
  energy_price: number;        // ct/kWh
  producer_price: number;      // ct/kWh
  use_vat: boolean;
  vat_pct: number;
  meter_fee_eur: number;
  free_kwh: number;
  discount_pct: number;
  participation_fee_eur: number;
  billing_period: string;      // monthly|quarterly|semiannual|annual
  invoice_number_prefix: string;
  invoice_number_digits: number;
  invoice_number_start: number;
  invoice_pre_text: string;
  invoice_post_text: string;
  invoice_footer_text: string;
  logo_path: string;
  generate_credit_notes: boolean;
  credit_note_number_prefix: string;
  credit_note_number_digits: number;
  iban: string;
  bic: string;
  sepa_creditor_id: string;
  sepa_pre_notification_days?: number; // SEPA pre-notification period in days (default 14)
  // EDA communication settings
  eda_marktpartner_id: string;
  eda_netzbetreiber_id: string;
  eda_transition_date?: string; // ISO date string YYYY-MM-DD
  // Accounting / DATEV
  accounting_revenue_account: number;
  accounting_expense_account: number;
  accounting_debitor_prefix: number;
  datev_consultant_nr: string;
  datev_client_nr: string;
  // Rechnungssteller address (§11 UStG)
  strasse: string;
  plz: string;
  ort: string;
  uid_nummer: string;
  gruendungsdatum?: string; // ISO date string YYYY-MM-DD
  onboarding_contract_text?: string;
  is_demo?: boolean;
  // Auto-billing
  auto_billing_enabled?: boolean;
  auto_billing_day_of_month?: number; // 1–28, 0 = not set
  auto_billing_period?: string;       // "monthly"|"quarterly"
  auto_billing_last_run_at?: string;  // ISO datetime
  // Gap alert
  gap_alert_enabled?: boolean;
  gap_alert_threshold_days?: number;  // default 5
  // Member portal
  portal_show_full_energy?: boolean;
  created_at?: string;
  // Per-EEG EDA credentials (passwords omitted from API responses)
  eda_imap_host?: string;
  eda_imap_user?: string;
  eda_smtp_host?: string;
  eda_smtp_user?: string;
  eda_smtp_from?: string;
  // Per-EEG invoice SMTP credentials (passwords omitted from API responses)
  smtp_host?: string;
  smtp_user?: string;
  smtp_from?: string;
}

export interface EDAProcess {
  id: string;
  eeg_id: string;
  meter_point_id?: string;
  process_type: string;         // EC_REQ_ONL | EC_REQ_OFF | CM_REV_SP | EC_PRTFACT_CHG | EC_REQ_PT
  status: string;               // pending | sent | first_confirmed | confirmed | completed | rejected | error
  conversation_id: string;
  zaehlpunkt: string;
  valid_from?: string;
  participation_factor?: number;
  share_type?: string;
  initiated_at: string;
  deadline_at?: string;
  completed_at?: string;
  error_msg?: string;
  created_at: string;
  member_name?: string;
}

export interface CreateEEGRequest {
  gemeinschaft_id: string;
  netzbetreiber: string;
  name: string;
  energy_price: number;
  producer_price?: number;
  use_vat?: boolean;
  vat_pct?: number;
  meter_fee_eur?: number;
  free_kwh?: number;
  discount_pct?: number;
  participation_fee_eur?: number;
  billing_period?: string;
}

export interface UpdateEEGRequest {
  name?: string;
  netzbetreiber?: string;
  energy_price: number;
  producer_price: number;
  use_vat: boolean;
  vat_pct: number;
  meter_fee_eur: number;
  free_kwh: number;
  discount_pct: number;
  participation_fee_eur: number;
  billing_period: string;
  invoice_number_prefix?: string;
  invoice_number_digits?: number;
  invoice_number_start?: number;
  invoice_pre_text?: string;
  invoice_post_text?: string;
  invoice_footer_text?: string;
  generate_credit_notes?: boolean;
  credit_note_number_prefix?: string;
  credit_note_number_digits?: number;
  iban?: string;
  bic?: string;
  sepa_creditor_id?: string;
  sepa_pre_notification_days?: number;
  gemeinschaft_id?: string;
  eda_marktpartner_id?: string;
  eda_netzbetreiber_id?: string;
  eda_transition_date?: string;
  gruendungsdatum?: string;
  accounting_revenue_account?: number;
  accounting_expense_account?: number;
  accounting_debitor_prefix?: number;
  datev_consultant_nr?: string;
  datev_client_nr?: string;
  strasse?: string;
  plz?: string;
  ort?: string;
  uid_nummer?: string;
  onboarding_contract_text?: string;
  // Per-EEG EDA credentials (send only when changing; empty = keep existing)
  eda_imap_host?: string;
  eda_imap_user?: string;
  eda_imap_password?: string;
  eda_smtp_host?: string;
  eda_smtp_user?: string;
  eda_smtp_password?: string;
  eda_smtp_from?: string;
  // Per-EEG invoice SMTP credentials
  smtp_host?: string;
  smtp_user?: string;
  smtp_password?: string;
  smtp_from?: string;
  // Auto-billing
  auto_billing_enabled?: boolean;
  auto_billing_day_of_month?: number;
  auto_billing_period?: string;
  // Gap alert
  gap_alert_enabled?: boolean;
  gap_alert_threshold_days?: number;
  // Member portal
  portal_show_full_energy?: boolean;
}

export interface MeterPoint {
  id: string;
  meter_id: string;
  direction: string;
  generation_type?: string; // PV | Windkraft | Wasserkraft
  name?: string;
  participation_factor?: number; // 0.0001–100; from confirmed EC_REQ_ONL EDA process
  factor_valid_from?: string;    // ISO date; valid_from of the EDA process that set the factor
  registriert_seit?: string; // ISO date, NB-confirmed activation date
  abgemeldet_am?: string;    // ISO date, NB-confirmed deregistration date
  anmeldung_status?: string; // latest EC_REQ_ONL process status
  abmeldung_status?: string; // latest CM_REV_SP process status
  gap_alert_sent_at?: string; // ISO datetime when gap alert was sent; set = no recent readings
}

export interface MeterPointGap {
  id: string;
  zaehlpunkt: string;
  member_id: string;
  member_name: string;
  gap_alert_sent_at: string;
  last_reading_at?: string;
}

export interface Member {
  id: string;
  name: string;
  name1?: string;
  name2?: string;
  email?: string;
  member_number?: string;
  mitglieds_nr?: string;
  iban?: string;
  strasse?: string;
  plz?: string;
  ort?: string;
  business_role?: string;
  uid_nummer?: string;
  // null = inherit from EEG, true/false = explicit override
  use_vat?: boolean | null;
  vat_pct?: number | null;
  // ACTIVE | REGISTERED | NEW | INACTIVE
  status?: string;
  // ISO date strings (YYYY-MM-DD)
  beitritts_datum?: string | null;
  austritts_datum?: string | null;
  meter_points: MeterPoint[];
}

export async function searchMembers(
  token: string,
  eegId: string,
  q?: string,
  status?: string
): Promise<Member[]> {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (status) params.set("status", status);
  const qs = params.toString() ? `?${params}` : "";
  return apiFetch<Member[]>(`/eegs/${eegId}/members${qs}`, token);
}

export interface ImportResult {
  members_imported?: number;
  meter_points_imported?: number;
  rows_imported?: number;
  message?: string;
  errors?: string[];
}

export interface BillingRunRequest {
  period_start: string; // YYYY-MM-DD
  period_end: string;   // YYYY-MM-DD
}

export interface BillingRun {
  id: string;
  eeg_id: string;
  period_start: string;
  period_end: string;
  status: string;
  invoice_count: number;
  total_amount: number;
  created_at?: string;
}

export interface Invoice {
  id: string;
  member_id: string;
  eeg_id: string;
  period_start: string;
  period_end: string;
  total_kwh: number;
  total_amount: number;
  consumption_kwh?: number;
  generation_kwh?: number;
  pdf_path?: string;
  storno_pdf_path?: string;
  sent_at?: string;
  status: string;
  document_type?: string; // "invoice" | "credit_note"
  invoice_number?: number;
  billing_run_id?: string;
  created_at?: string;
  // SEPA return (Rücklastschrift) tracking
  sepa_return_at?: string;
  sepa_return_reason?: string;
  sepa_return_note?: string;
}

export interface ApiError {
  message: string;
  status: number;
}

function getApiBaseUrl(): string {
  // Server-side: use internal URL if available
  if (typeof window === "undefined") {
    return process.env.API_INTERNAL_URL || process.env.API_URL_INTERNAL || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8101";
  }
  // Client-side: use public URL
  return process.env.NEXT_PUBLIC_API_URL || "http://localhost:8101";
}

async function apiFetch<T>(
  path: string,
  token: string,
  options: RequestInit = {}
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = `${baseUrl}/api/v1${path}`;

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    ...(options.headers as Record<string, string>),
  };

  // Only set Content-Type to JSON if not sending FormData
  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(url, {
    ...options,
    headers,
  });

  if (!res.ok) {
    // Client-side 401: redirect to sign-in (session/token expired)
    if (res.status === 401 && typeof window !== "undefined") {
      window.location.href = "/auth/signin";
    }
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      message = body.message || body.error || message;
    } catch {
      // ignore parse error
    }
    const err: ApiError = { message, status: res.status };
    throw err;
  }

  // 204 No Content
  if (res.status === 204) {
    return {} as T;
  }

  return res.json() as Promise<T>;
}

// EEG endpoints

export async function listEEGs(token: string): Promise<EEG[]> {
  return apiFetch<EEG[]>("/eegs", token);
}

export async function createEEG(
  token: string,
  data: CreateEEGRequest
): Promise<EEG> {
  return apiFetch<EEG>("/eegs", token, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getEEG(token: string, eegId: string): Promise<EEG> {
  return apiFetch<EEG>(`/eegs/${eegId}`, token);
}

export async function updateEEG(token: string, eegId: string, data: UpdateEEGRequest): Promise<EEG> {
  return apiFetch<EEG>(`/eegs/${eegId}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export interface EEGStats {
  member_count: number;
  meter_point_count: number;
  invoice_count: number;
  billing_run_count: number;
  total_kwh: number;
  total_revenue: number;
  last_billing_run?: string;
}

export async function getStats(token: string, eegId: string): Promise<EEGStats> {
  return apiFetch<EEGStats>(`/eegs/${eegId}/stats`, token);
}

export async function listGapAlerts(token: string, eegId: string): Promise<MeterPointGap[]> {
  return apiFetch<MeterPointGap[]>(`/eegs/${eegId}/gap-alerts`, token);
}

export interface EEGMeterParticipation {
  id: string;
  eeg_id: string;
  meter_point_id: string;
  participation_factor: number; // 0.0001–100
  share_type: string;           // GC | RC_R | RC_L | CC
  valid_from: string;           // YYYY-MM-DD
  valid_until?: string;         // YYYY-MM-DD | undefined (open-ended)
  notes: string;
  created_at: string;
}

export async function listParticipations(token: string, eegId: string): Promise<EEGMeterParticipation[]> {
  return apiFetch<EEGMeterParticipation[]>(`/eegs/${eegId}/participations`, token);
}

// Member endpoints

export async function listMembers(
  token: string,
  eegId: string,
  stichtag?: string
): Promise<Member[]> {
  const qs = stichtag ? `?stichtag=${encodeURIComponent(stichtag)}` : "";
  return apiFetch<Member[]>(`/eegs/${eegId}/members${qs}`, token);
}

// Import endpoints

export async function importStammdaten(
  token: string,
  eegId: string,
  file: File
): Promise<ImportResult> {
  const formData = new FormData();
  formData.append("file", file);
  return apiFetch<ImportResult>(`/eegs/${eegId}/import/stammdaten`, token, {
    method: "POST",
    body: formData,
  });
}

export async function importEnergiedaten(
  token: string,
  eegId: string,
  file: File
): Promise<ImportResult> {
  const formData = new FormData();
  formData.append("file", file);
  return apiFetch<ImportResult>(`/eegs/${eegId}/import/energiedaten`, token, {
    method: "POST",
    body: formData,
  });
}

// Billing endpoints

export async function runBilling(
  token: string,
  eegId: string,
  request: BillingRunRequest
): Promise<{ billing_run: BillingRun; invoices_created: number }> {
  return apiFetch(`/eegs/${eegId}/billing/run`, token, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

export async function listBillingRuns(
  token: string,
  eegId: string
): Promise<BillingRun[]> {
  return apiFetch<BillingRun[]>(`/eegs/${eegId}/billing/runs`, token);
}

export async function listInvoicesByBillingRun(
  token: string,
  eegId: string,
  runId: string
): Promise<Invoice[]> {
  return apiFetch<Invoice[]>(`/eegs/${eegId}/billing/runs/${runId}/invoices`, token);
}

export async function listInvoices(
  token: string,
  eegId: string
): Promise<Invoice[]> {
  return apiFetch<Invoice[]>(`/eegs/${eegId}/invoices`, token);
}

export async function listSepaReturns(
  token: string,
  eegId: string
): Promise<Invoice[]> {
  return apiFetch<Invoice[]>(`/eegs/${eegId}/invoices?sepa_returned=true`, token);
}

export async function countSepaReturns(
  token: string,
  eegId: string
): Promise<number> {
  const invoices = await listSepaReturns(token, eegId);
  return invoices.length;
}

export async function setSepaReturn(
  token: string,
  eegId: string,
  invoiceId: string,
  data: { return_at?: string; reason?: string; note?: string }
): Promise<Invoice> {
  return apiFetch<Invoice>(`/eegs/${eegId}/invoices/${invoiceId}/sepa-return`, token, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export interface Camt054ImportResult {
  matched: number;
  not_found: number;
  already_returned: number;
}

export async function importCamt054(
  token: string,
  eegId: string,
  file: File
): Promise<Camt054ImportResult> {
  const formData = new FormData();
  formData.append("file", file);
  return apiFetch<Camt054ImportResult>(`/eegs/${eegId}/sepa/camt054`, token, {
    method: "POST",
    body: formData,
  });
}

// Member CRUD

export interface CreateMemberRequest {
  mitglieds_nr?: string;
  name1: string;
  name2?: string;
  email?: string;
  iban?: string;
  strasse?: string;
  plz?: string;
  ort?: string;
  // "privat" | "kleinunternehmer" | "landwirt_pauschaliert" | "landwirt" |
  // "unternehmen" | "gemeinde_bga" | "gemeinde_hoheitlich"
  business_role?: string;
  // UID-Nummer (VAT ID): determines Reverse Charge on credit notes
  uid_nummer?: string;
  // reserved for future manual override on credit notes (NOT applied to consumer invoices)
  use_vat?: boolean | null;
  vat_pct?: number | null;
  // "ACTIVE" | "REGISTERED" | "INACTIVE"
  status?: string;
  // ISO date strings YYYY-MM-DD (optional)
  beitritts_datum?: string;
  austritts_datum?: string;
}

export async function createMember(
  token: string,
  eegId: string,
  data: CreateMemberRequest
): Promise<Member> {
  return apiFetch<Member>(`/eegs/${eegId}/members`, token, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getMember(
  token: string,
  eegId: string,
  memberId: string
): Promise<Member> {
  return apiFetch<Member>(`/eegs/${eegId}/members/${memberId}`, token);
}

export async function updateMember(
  token: string,
  eegId: string,
  memberId: string,
  data: CreateMemberRequest
): Promise<Member> {
  return apiFetch<Member>(`/eegs/${eegId}/members/${memberId}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteMember(
  token: string,
  eegId: string,
  memberId: string
): Promise<void> {
  await apiFetch<void>(`/eegs/${eegId}/members/${memberId}`, token, {
    method: "DELETE",
  });
}

// Meter Points

export interface MeterPointFull {
  id: string;
  member_id: string;
  eeg_id: string;
  zaehlpunkt: string;
  energierichtung: string;
  verteilungsmodell: string;
  zugeteilte_menge_pct: number;
  status: string;
  registriert_seit?: string;
  generation_type?: string; // PV | Windkraft | Wasserkraft
  notes?: string;
}

export interface CreateMeterPointRequest {
  zaehlpunkt: string;
  energierichtung: string; // "CONSUMPTION" | "GENERATION"
  verteilungsmodell?: string; // "DYNAMIC" | "STATIC"
  zugeteilte_menge_pct?: number;
  status?: string;
  registriert_seit?: string; // YYYY-MM-DD
  generation_type?: string;
  notes?: string;
}

export async function createMeterPoint(
  token: string,
  eegId: string,
  memberId: string,
  data: CreateMeterPointRequest
): Promise<MeterPointFull> {
  return apiFetch<MeterPointFull>(
    `/eegs/${eegId}/members/${memberId}/meter-points`,
    token,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  );
}

export async function updateMeterPoint(
  token: string,
  eegId: string,
  meterPointId: string,
  data: CreateMeterPointRequest
): Promise<MeterPointFull> {
  return apiFetch<MeterPointFull>(
    `/eegs/${eegId}/meter-points/${meterPointId}`,
    token,
    {
      method: "PUT",
      body: JSON.stringify(data),
    }
  );
}

export async function getMeterPoint(
  token: string,
  eegId: string,
  meterPointId: string
): Promise<MeterPointFull> {
  return apiFetch<MeterPointFull>(
    `/eegs/${eegId}/meter-points/${meterPointId}`,
    token
  );
}

export async function deleteMeterPoint(
  token: string,
  eegId: string,
  meterPointId: string
): Promise<void> {
  await apiFetch<void>(`/eegs/${eegId}/meter-points/${meterPointId}`, token, {
    method: "DELETE",
  });
}

// Invoice status

export async function updateInvoiceStatus(
  token: string,
  eegId: string,
  invoiceId: string,
  status: string
): Promise<void> {
  await apiFetch<void>(`/eegs/${eegId}/invoices/${invoiceId}/status`, token, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export interface SendAllResult {
  sent: number;
  skipped: number;
  failed: number;
  errors?: string[];
}

export async function sendAllInvoices(
  token: string,
  eegId: string
): Promise<SendAllResult> {
  return apiFetch<SendAllResult>(`/eegs/${eegId}/invoices/send-all`, token, {
    method: "POST",
  });
}

export async function sendAllInvoicesByRun(
  token: string,
  eegId: string,
  runId: string
): Promise<SendAllResult> {
  return apiFetch<SendAllResult>(
    `/eegs/${eegId}/billing/runs/${runId}/send-all`,
    token,
    { method: "POST" }
  );
}

export async function finalizeBillingRun(
  token: string,
  eegId: string,
  runId: string
): Promise<BillingRun> {
  return apiFetch<BillingRun>(
    `/eegs/${eegId}/billing/runs/${runId}/finalize`,
    token,
    { method: "POST" }
  );
}

export async function deleteBillingRun(
  token: string,
  eegId: string,
  runId: string
): Promise<void> {
  await apiFetch<void>(
    `/eegs/${eegId}/billing/runs/${runId}`,
    token,
    { method: "DELETE" }
  );
}

export async function cancelBillingRun(
  token: string,
  eegId: string,
  runId: string
): Promise<BillingRun> {
  return apiFetch<BillingRun>(
    `/eegs/${eegId}/billing/runs/${runId}/cancel`,
    token,
    { method: "POST" }
  );
}

export async function resendInvoice(
  token: string,
  eegId: string,
  invoiceId: string
): Promise<void> {
  await apiFetch<void>(`/eegs/${eegId}/invoices/${invoiceId}/resend`, token, {
    method: "POST",
  });
}

// OeMAG market prices

export interface OemagMonthPrice {
  month: number;      // 1–12
  pv_price: number;   // ct/kWh
  wind_price: number; // ct/kWh
}

export interface OemagYearPrices {
  year: number;
  prices: OemagMonthPrice[];
  static: boolean;
}

export interface OemagMarktpreisResult {
  years: OemagYearPrices[];
  scraped_at: string;
}

export interface OemagNewMonth {
  year: number;
  price: OemagMonthPrice;
}

export interface OemagRefreshResult {
  all: OemagYearPrices[];
  new_months: OemagNewMonth[];
  scraped_at: string;
}

export async function getOemagMarktpreis(token: string): Promise<OemagMarktpreisResult> {
  return apiFetch<OemagMarktpreisResult>("/oemag/marktpreis", token);
}

export async function refreshOemagMarktpreis(token: string): Promise<OemagRefreshResult> {
  return apiFetch<OemagRefreshResult>("/oemag/refresh", token, { method: "POST" });
}

export async function syncOemagPrice(
  token: string,
  eegId: string,
  priceType: "pv" | "wind",
  target: "producer_price" | "energy_price"
): Promise<{ month: number; price: number; target: string; price_type: string }> {
  return apiFetch(`/eegs/${eegId}/oemag/sync`, token, {
    method: "POST",
    body: JSON.stringify({ price_type: priceType, target }),
  });
}

// Stats

export interface EEGStats {
  member_count: number;
  invoice_count: number;
  total_kwh: number;
  last_billing_run?: string;
}

export async function getEEGStats(
  token: string,
  eegId: string
): Promise<EEGStats> {
  return apiFetch<EEGStats>(`/eegs/${eegId}/stats`, token);
}

// EDA Messages

export interface EDAMessage {
  id: string;
  message_id: string;
  direction: string;
  process: string;
  message_type: string;
  subject: string;
  body?: string;
  from_address?: string;
  to_address?: string;
  status: string;
  error_msg?: string;
  processed_at?: string;
  created_at: string;
}

export interface EDAMessageListResponse {
  messages: EDAMessage[];
  total_count: number;
  limit: number;
  offset: number;
}

export async function listEDAMessages(
  token: string,
  eegId: string,
  options?: { limit?: number; offset?: number }
): Promise<EDAMessageListResponse> {
  const params = new URLSearchParams();
  if (options?.limit) params.set("limit", String(options.limit));
  if (options?.offset) params.set("offset", String(options.offset));
  const qs = params.toString() ? `?${params.toString()}` : "";
  return apiFetch<EDAMessageListResponse>(`/eegs/${eegId}/eda/messages${qs}`, token);
}

export async function listEDAProcesses(
  token: string,
  eegId: string
): Promise<EDAProcess[]> {
  return apiFetch<EDAProcess[]>(`/eegs/${eegId}/eda/processes`, token);
}

// EDA Errors (dead-letter log)
export interface EDAError {
  id: string;
  eeg_id?: string;
  direction: string;
  message_type: string;
  subject: string;
  raw_content: string;
  error_msg: string;
  created_at: string;
}

export async function listEDAErrors(
  token: string,
  eegId: string
): Promise<EDAError[]> {
  return apiFetch<EDAError[]>(`/eegs/${eegId}/eda/errors`, token);
}

// EDA Worker status
export interface EDAWorkerStatus {
  transport_mode: string;
  last_poll_at?: string;
  last_error: string;
  updated_at: string;
}

export async function getEDAWorkerStatus(token: string): Promise<EDAWorkerStatus> {
  return apiFetch<EDAWorkerStatus>("/eda/worker-status", token);
}

// Reports

export interface MonthlyEnergyRow {
  month: string;
  consumption_kwh: number;
  generation_kwh: number;
  revenue: number;
  payouts: number;
}

export interface MemberStat {
  member_id: string;
  consumption_kwh: number;       // EEG share consumed (wh_self)
  generation_kwh: number;        // EEG share fed in (wh_community)
  consumption_total_kwh: number; // total consumption (wh_total); 0 in billed mode
  generation_total_kwh: number;  // total generation (wh_total); 0 in billed mode
  total_amount: number;
  invoice_count: number;
}

export async function getMonthlyEnergy(
  token: string,
  eegId: string,
  year: number
): Promise<MonthlyEnergyRow[]> {
  return apiFetch<MonthlyEnergyRow[]>(`/eegs/${eegId}/reports/energy?year=${year}`, token);
}

export async function getMemberStats(
  token: string,
  eegId: string,
  from?: string,
  to?: string
): Promise<MemberStat[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const qs = params.toString() ? `?${params}` : "";
  return apiFetch<MemberStat[]>(`/eegs/${eegId}/reports/members${qs}`, token);
}

export interface EnergySummaryRow {
  period: string;
  wh_self: number;               // EEG share consumed (kWh)
  wh_community: number;          // EEG share fed in by generators (kWh)
  wh_total_consumption: number;  // total consumption (kWh)
  wh_total_generation: number;   // total generation (kWh)
  wh_restbedarf: number;         // grid demand = wh_total_consumption - wh_self (kWh)
  wh_resteinspeisung: number;    // grid export  = wh_total_generation - wh_community (kWh)
}

// Admin: User management

export interface AdminUser {
  id: string;
  organization_id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

export async function listUsers(token: string): Promise<AdminUser[]> {
  return apiFetch<AdminUser[]>("/admin/users", token);
}

export async function getUser(token: string, userId: string): Promise<AdminUser> {
  return apiFetch<AdminUser>(`/admin/users/${userId}`, token);
}

export async function createUser(
  token: string,
  data: { name: string; email: string; password: string; role: string }
): Promise<AdminUser> {
  return apiFetch<AdminUser>("/admin/users", token, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateUser(
  token: string,
  userId: string,
  data: { name: string; email: string; role: string; password?: string }
): Promise<AdminUser> {
  return apiFetch<AdminUser>(`/admin/users/${userId}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteUser(token: string, userId: string): Promise<void> {
  await apiFetch<void>(`/admin/users/${userId}`, token, { method: "DELETE" });
}

export async function getUserEEGs(token: string, userId: string): Promise<string[]> {
  return apiFetch<string[]>(`/admin/users/${userId}/eegs`, token);
}

export async function setUserEEGs(
  token: string,
  userId: string,
  eegIds: string[]
): Promise<void> {
  await apiFetch<void>(`/admin/users/${userId}/eegs`, token, {
    method: "PUT",
    body: JSON.stringify(eegIds),
  });
}

// ── Tariff schedules ──────────────────────────────────────────────────────────

export interface TariffEntry {
  id: string;
  schedule_id: string;
  valid_from: string;
  valid_until: string;
  energy_price: number;   // ct/kWh
  producer_price: number; // ct/kWh
  created_at: string;
}

export interface TariffSchedule {
  id: string;
  eeg_id: string;
  name: string;
  granularity: "annual" | "monthly" | "daily" | "quarter_hour";
  is_active: boolean;
  entry_count?: number;
  entries?: TariffEntry[];
  created_at: string;
}

export async function listTariffs(token: string, eegId: string): Promise<TariffSchedule[]> {
  return apiFetch<TariffSchedule[]>(`/eegs/${eegId}/tariffs`, token);
}

export async function getTariff(token: string, eegId: string, scheduleId: string): Promise<TariffSchedule> {
  return apiFetch<TariffSchedule>(`/eegs/${eegId}/tariffs/${scheduleId}`, token);
}

export async function createTariff(token: string, eegId: string, data: { name: string; granularity: string }): Promise<TariffSchedule> {
  return apiFetch<TariffSchedule>(`/eegs/${eegId}/tariffs`, token, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateTariff(token: string, eegId: string, scheduleId: string, data: { name: string; granularity: string }): Promise<TariffSchedule> {
  return apiFetch<TariffSchedule>(`/eegs/${eegId}/tariffs/${scheduleId}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteTariff(token: string, eegId: string, scheduleId: string): Promise<void> {
  return apiFetch<void>(`/eegs/${eegId}/tariffs/${scheduleId}`, token, { method: "DELETE" });
}

export async function activateTariff(token: string, eegId: string, scheduleId: string): Promise<void> {
  return apiFetch<void>(`/eegs/${eegId}/tariffs/${scheduleId}/activate`, token, { method: "POST" });
}

export async function deactivateTariff(token: string, eegId: string, scheduleId: string): Promise<void> {
  return apiFetch<void>(`/eegs/${eegId}/tariffs/${scheduleId}/activate`, token, { method: "DELETE" });
}

// ── Onboarding ────────────────────────────────────────────────────────────────

export interface OnboardingMeterPoint {
  zaehlpunkt: string;
  direction: string;
}

export interface OnboardingRequest {
  id: string;
  eeg_id: string;
  status: string;
  name1: string;
  name2: string;
  email: string;
  phone: string;
  strasse: string;
  plz: string;
  ort: string;
  iban: string;
  bic: string;
  member_type: string;
  meter_points: OnboardingMeterPoint[];
  contract_accepted_at?: string;
  admin_notes: string;
  converted_member_id?: string;
  created_at: string;
  updated_at: string;
}

export async function listOnboardingRequests(
  token: string,
  eegId: string
): Promise<OnboardingRequest[]> {
  return apiFetch<OnboardingRequest[]>(`/eegs/${eegId}/onboarding`, token);
}

export async function updateOnboardingStatus(
  token: string,
  eegId: string,
  requestId: string,
  status: string,
  notes: string
): Promise<OnboardingRequest> {
  return apiFetch<OnboardingRequest>(
    `/eegs/${eegId}/onboarding/${requestId}`,
    token,
    {
      method: "PATCH",
      body: JSON.stringify({ status, admin_notes: notes }),
    }
  );
}

export async function setTariffEntries(
  token: string, eegId: string, scheduleId: string,
  entries: Array<{ valid_from: string; valid_until: string; energy_price: number; producer_price: number }>
): Promise<{ saved: number }> {
  return apiFetch<{ saved: number }>(`/eegs/${eegId}/tariffs/${scheduleId}/entries`, token, {
    method: "PUT",
    body: JSON.stringify(entries),
  });
}
