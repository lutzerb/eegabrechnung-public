package xml

import (
	"strings"
	"testing"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/eda/edaversion"
)

// TestEcmpListSchemaHistory verifies the schema cutover table resolves to
// 01.10 before 2026-10-05 and 01.20 from that date onward. This is the same
// mechanism BuildECMPList uses internally via time.Now(); asserting the table
// itself (rather than BuildECMPList's live output) makes the pre/post cutover
// cases testable without depending on the real system clock.
func TestEcmpListSchemaHistory(t *testing.T) {
	before := edaversion.Cutover2026_10_05.Add(-time.Second)
	after := edaversion.Cutover2026_10_05

	got := edaversion.Resolve(ecmpListSchemaHistory, before)
	if got.SchemaVer != "01.10" || !strings.HasSuffix(got.NS, "/ecmplist/01p10") {
		t.Errorf("before cutover: got %+v, want schema 01.10", got)
	}

	got = edaversion.Resolve(ecmpListSchemaHistory, after)
	if got.SchemaVer != "01.20" || !strings.HasSuffix(got.NS, "/ecmplist/01p20") {
		t.Errorf("at/after cutover: got %+v, want schema 01.20", got)
	}
}

// TestBuildECMPListCurrentSchema exercises BuildECMPList end-to-end against
// the current system clock. As of this test's writing (pre-2026-10-05) it
// must still emit the 01.10 namespace without a DataType element.
func TestBuildECMPListCurrentSchema(t *testing.T) {
	if !time.Now().Before(edaversion.Cutover2026_10_05) {
		t.Skip("system clock is at/after the 2026-10-05 cutover; this pre-cutover assertion no longer applies")
	}

	out, err := BuildECMPList(ECMPListParams{
		From:            "AT002000",
		To:              "RC105970",
		MessageID:       "11111111-1111-1111-1111-111111111111",
		ConversationID:  "22222222-2222-2222-2222-222222222222",
		ECID:            "AT00200000000RC105970000000001289",
		ECType:          "RC_R",
		ECDisModel:      "D",
		MessageCode:     "ANFORDERUNG_CPF",
		MeteringPoint:   "AT0020000000000000000000100266304",
		DateFrom:        time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EnergyDirection: "CONSUMPTION",
		ECPartFact:      50,
	})
	if err != nil {
		t.Fatalf("BuildECMPList: %v", err)
	}
	if !strings.Contains(out, "ecmplist/01p10") {
		t.Errorf("expected 01.10 namespace before cutover, got: %s", out)
	}
	if strings.Contains(out, "cp:DataType") {
		t.Errorf("DataType must not be rendered before cutover, got: %s", out)
	}
}

// TestBuildECMPListRequiresDataTypeAfterCutover documents that BuildECMPList
// will require DataType once the system clock reaches the 2026-10-05 cutover.
// It only runs once that date has actually arrived, since BuildECMPList reads
// time.Now() internally and cannot be pointed at a fake future date.
func TestBuildECMPListRequiresDataTypeAfterCutover(t *testing.T) {
	if time.Now().Before(edaversion.Cutover2026_10_05) {
		t.Skip("system clock is before the 2026-10-05 cutover; re-run this test on/after that date")
	}

	params := ECMPListParams{
		From:            "AT002000",
		To:              "RC105970",
		MessageID:       "11111111-1111-1111-1111-111111111111",
		ConversationID:  "22222222-2222-2222-2222-222222222222",
		ECID:            "AT00200000000RC105970000000001289",
		ECType:          "RC_R",
		ECDisModel:      "D",
		MessageCode:     "ANFORDERUNG_CPF",
		MeteringPoint:   "AT0020000000000000000000100266304",
		DateFrom:        time.Now(),
		EnergyDirection: "CONSUMPTION",
		ECPartFact:      50,
	}

	if _, err := BuildECMPList(params); err == nil {
		t.Error("expected error when DataType is missing after cutover, got nil")
	}

	params.DataType = "EnergyCommunityRegistration"
	out, err := BuildECMPList(params)
	if err != nil {
		t.Fatalf("BuildECMPList with DataType set: %v", err)
	}
	if !strings.Contains(out, "ecmplist/01p20") {
		t.Errorf("expected 01.20 namespace after cutover, got: %s", out)
	}
	if !strings.Contains(out, "<cp:DataType>EnergyCommunityRegistration</cp:DataType>") {
		t.Errorf("expected DataType element after cutover, got: %s", out)
	}
}
