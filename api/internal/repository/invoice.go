package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type InvoiceRepository struct {
	db *pgxpool.Pool
}

func NewInvoiceRepository(db *pgxpool.Pool) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

func (r *InvoiceRepository) nextNumberTx(ctx context.Context, tx pgx.Tx, eegID uuid.UUID, documentType string) (int, error) {
	var invoiceStart int
	if err := tx.QueryRow(ctx, `SELECT invoice_number_start FROM eegs WHERE id = $1 FOR UPDATE`, eegID).Scan(&invoiceStart); err != nil {
		return 0, fmt.Errorf("lock eeg for invoice number: %w", err)
	}

	var currentMax int
	q := `SELECT COALESCE(MAX(invoice_number), 0)
	      FROM invoices
	      WHERE eeg_id = $1 AND document_type = $2`
	if err := tx.QueryRow(ctx, q, eegID, documentType).Scan(&currentMax); err != nil {
		return 0, fmt.Errorf("query next invoice number: %w", err)
	}

	next := currentMax + 1
	if documentType == "invoice" && next < invoiceStart {
		next = invoiceStart
	}
	return next, nil
}

func (r *InvoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
	documentType := inv.DocumentType
	if documentType == "" {
		documentType = "invoice"
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin invoice create tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	n, err := r.nextNumberTx(ctx, tx, inv.EegID, documentType)
	if err != nil {
		return err
	}

	inv.DocumentType = documentType
	inv.InvoiceNumber = &n
	inv.Status = "draft"
	q := `INSERT INTO invoices
	        (member_id, eeg_id, period_start, period_end, total_kwh, total_amount,
	         net_amount, vat_amount, vat_pct_applied,
	         consumption_kwh, generation_kwh,
	         consumption_net_amount, generation_net_amount,
	         consumption_vat_pct, consumption_vat_amount, generation_vat_pct, generation_vat_amount,
	         pdf_path, invoice_number, status, billing_run_id, document_type)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	      RETURNING id, created_at`
	if err := tx.QueryRow(ctx, q,
		inv.MemberID, inv.EegID, inv.PeriodStart, inv.PeriodEnd,
		inv.TotalKwh, inv.TotalAmount,
		inv.NetAmount, inv.VatAmount, inv.VatPctApplied,
		inv.ConsumptionKwh, inv.GenerationKwh,
		inv.ConsumptionNetAmount, inv.GenerationNetAmount,
		inv.ConsumptionVatPct, inv.ConsumptionVatAmount, inv.GenerationVatPct, inv.GenerationVatAmount,
		inv.PdfPath, inv.InvoiceNumber, inv.Status, inv.BillingRunID, inv.DocumentType,
	).Scan(&inv.ID, &inv.CreatedAt); err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit invoice create tx: %w", err)
	}
	return nil
}

const invoiceCols = `id, member_id, eeg_id, period_start, period_end, total_kwh, total_amount,
	net_amount, vat_amount, vat_pct_applied,
	consumption_kwh, generation_kwh,
	consumption_net_amount, generation_net_amount,
	consumption_vat_pct, consumption_vat_amount, generation_vat_pct, generation_vat_amount,
	pdf_path, storno_pdf_path, sent_at, invoice_number, status, billing_run_id, document_type, created_at,
	sepa_return_at, sepa_return_reason, sepa_return_note`

func scanInvoice(row interface{ Scan(...any) error }, inv *domain.Invoice) error {
	var sepaReturnReason *string
	var sepaReturnNote *string
	err := row.Scan(
		&inv.ID, &inv.MemberID, &inv.EegID, &inv.PeriodStart, &inv.PeriodEnd,
		&inv.TotalKwh, &inv.TotalAmount,
		&inv.NetAmount, &inv.VatAmount, &inv.VatPctApplied,
		&inv.ConsumptionKwh, &inv.GenerationKwh,
		&inv.ConsumptionNetAmount, &inv.GenerationNetAmount,
		&inv.ConsumptionVatPct, &inv.ConsumptionVatAmount, &inv.GenerationVatPct, &inv.GenerationVatAmount,
		&inv.PdfPath, &inv.StornoPdfPath, &inv.SentAt, &inv.InvoiceNumber, &inv.Status, &inv.BillingRunID, &inv.DocumentType, &inv.CreatedAt,
		&inv.SepaReturnAt, &sepaReturnReason, &sepaReturnNote,
	)
	if sepaReturnReason != nil {
		inv.SepaReturnReason = *sepaReturnReason
	}
	if sepaReturnNote != nil {
		inv.SepaReturnNote = *sepaReturnNote
	}
	return err
}

func (r *InvoiceRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + ` FROM invoices WHERE eeg_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) SearchByEeg(ctx context.Context, eegID uuid.UUID, query string, limit int) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + `
	      FROM invoices
	      WHERE eeg_id = $1
	        AND invoice_number IS NOT NULL
	        AND invoice_number::text LIKE $2
	      ORDER BY created_at DESC
	      LIMIT $3`
	rows, err := r.db.Query(ctx, q, eegID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) ListByBillingRun(ctx context.Context, billingRunID uuid.UUID) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + ` FROM invoices WHERE billing_run_id = $1 ORDER BY created_at`
	rows, err := r.db.Query(ctx, q, billingRunID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// ListByEegAndPeriod returns all invoices for an EEG whose created_at falls within [from, to].
func (r *InvoiceRepository) ListByEegAndPeriod(ctx context.Context, eegID uuid.UUID, from, to time.Time) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + `
	      FROM invoices
	      WHERE eeg_id = $1 AND created_at >= $2 AND created_at <= $3
	      ORDER BY created_at`
	rows, err := r.db.Query(ctx, q, eegID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) UpdatePdfPath(ctx context.Context, id uuid.UUID, path string) error {
	q := `UPDATE invoices SET pdf_path = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, path)
	if err != nil {
		return fmt.Errorf("update pdf_path: %w", err)
	}
	return nil
}

func (r *InvoiceRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE invoices SET sent_at = NOW(), status = 'sent' WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	return nil
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + ` FROM invoices WHERE id = $1`
	var inv domain.Invoice
	if err := scanInvoice(r.db.QueryRow(ctx, q, id), &inv); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &inv, nil
}

// ListByMember returns all non-cancelled invoices for a member within an EEG, ordered by period_start DESC.
func (r *InvoiceRepository) ListByMember(ctx context.Context, eegID, memberID uuid.UUID) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + ` FROM invoices WHERE eeg_id = $1 AND member_id = $2 AND status NOT IN ('draft', 'cancelled') ORDER BY period_start DESC`
	rows, err := r.db.Query(ctx, q, eegID, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// SetSepaReturn records a SEPA return (Rücklastschrift) on an invoice.
func (r *InvoiceRepository) SetSepaReturn(ctx context.Context, id uuid.UUID, returnAt time.Time, reason, note string) error {
	q := `UPDATE invoices SET sepa_return_at = $2, sepa_return_reason = $3, sepa_return_note = $4 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, returnAt, reason, note)
	if err != nil {
		return fmt.Errorf("set sepa return: %w", err)
	}
	return nil
}

// ClearSepaReturn removes a previously recorded SEPA return from an invoice.
func (r *InvoiceRepository) ClearSepaReturn(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE invoices SET sepa_return_at = NULL, sepa_return_reason = NULL, sepa_return_note = NULL WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("clear sepa return: %w", err)
	}
	return nil
}

// ListByEegWithReturns returns all invoices for an EEG that have a SEPA return recorded.
func (r *InvoiceRepository) ListByEegWithReturns(ctx context.Context, eegID uuid.UUID) ([]domain.Invoice, error) {
	q := `SELECT ` + invoiceCols + ` FROM invoices WHERE eeg_id = $1 AND sepa_return_at IS NOT NULL ORDER BY sepa_return_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// CountSepaReturns returns the number of invoices with SEPA returns for an EEG.
func (r *InvoiceRepository) CountSepaReturns(ctx context.Context, eegID uuid.UUID) (int, error) {
	var count int
	q := `SELECT COUNT(*) FROM invoices WHERE eeg_id = $1 AND sepa_return_at IS NOT NULL`
	if err := r.db.QueryRow(ctx, q, eegID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sepa returns: %w", err)
	}
	return count, nil
}

// UpdateStornoPdfPath stores the storno PDF path for an invoice.
func (r *InvoiceRepository) UpdateStornoPdfPath(ctx context.Context, id uuid.UUID, path string) error {
	_, err := r.db.Exec(ctx, `UPDATE invoices SET storno_pdf_path=$2 WHERE id=$1`, id, path)
	return err
}

// UpdateStatus sets the status field for an invoice.
func (r *InvoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	q := `UPDATE invoices SET status=$2 WHERE id=$1`
	_, err := r.db.Exec(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
