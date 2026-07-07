package handler

import "testing"

// The billing CAMT.053 matcher must see the credit/debit direction and the
// per-transaction amount: a Rücklastschrift (DBIT) carries the same EndToEndId
// as the original collection (CRDT), and in a batch entry (Sammler) the Ntry
// amount is the batch total, not the invoice amount.
func TestExtractCAMT053EndToEndIds(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Ntry>
        <Amt Ccy="EUR">42.50</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2026-07-01</Dt></BookgDt>
        <NtryDtls>
          <TxDtls>
            <Refs><EndToEndId>aaaaaaaabbbbccccddddeeeeeeeeeeee</EndToEndId></Refs>
          </TxDtls>
        </NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">42.50</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2026-07-03</Dt></BookgDt>
        <NtryDtls>
          <TxDtls>
            <Refs><EndToEndId>aaaaaaaabbbbccccddddeeeeeeeeeeee</EndToEndId></Refs>
          </TxDtls>
        </NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">100.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2026-07-02</Dt></BookgDt>
        <NtryDtls>
          <TxDtls>
            <Refs><EndToEndId>11111111222233334444555555555555</EndToEndId></Refs>
            <Amt Ccy="EUR">60.00</Amt>
          </TxDtls>
          <TxDtls>
            <Refs><EndToEndId>66666666777788889999aaaaaaaaaaaa</EndToEndId></Refs>
            <Amt Ccy="EUR">40.00</Amt>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`)

	matches := extractCAMT053EndToEndIds(xmlData)
	if matches == nil {
		t.Fatal("expected parse to succeed, got nil")
	}
	if len(matches) != 4 {
		t.Fatalf("expected 4 tx matches, got %d", len(matches))
	}

	// Single-tx entry: direction + Ntry amount fallback
	if matches[0].cdtDbtInd != "CRDT" {
		t.Errorf("tx0 direction: expected CRDT, got %q", matches[0].cdtDbtInd)
	}
	if matches[0].amount != 42.50 {
		t.Errorf("tx0 amount: expected 42.50 (Ntry fallback), got %v", matches[0].amount)
	}

	// The Rücklastschrift entry must come out as DBIT so the matcher can skip it
	if matches[1].cdtDbtInd != "DBIT" {
		t.Errorf("tx1 direction: expected DBIT, got %q", matches[1].cdtDbtInd)
	}

	// Batch entry: per-TxDtls amounts, never the batch total
	if matches[2].amount != 60.00 {
		t.Errorf("tx2 amount: expected 60.00 (TxDtls), got %v", matches[2].amount)
	}
	if matches[3].amount != 40.00 {
		t.Errorf("tx3 amount: expected 40.00 (TxDtls), got %v", matches[3].amount)
	}
	if matches[2].cdtDbtInd != "CRDT" || matches[3].cdtDbtInd != "CRDT" {
		t.Errorf("batch tx directions: expected CRDT from Ntry fallback, got %q / %q",
			matches[2].cdtDbtInd, matches[3].cdtDbtInd)
	}

	if matches[1].bookingDate.Format("2006-01-02") != "2026-07-03" {
		t.Errorf("tx1 booking date: expected 2026-07-03, got %s", matches[1].bookingDate.Format("2006-01-02"))
	}
}
