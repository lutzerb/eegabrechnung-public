// pdfsample renders a standalone sample invoice PDF using GeneratePDFThemed
// (see internal/invoice/pdf_theme.go) with the shared invoice.SampleFixtureData()
// fixture — no database/Docker stack required.
//
// Usage: go run ./cmd/pdfsample [output-path]
// Default output path: /tmp/invoice_theme_prototype.pdf
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/lutzerb/eegabrechnung/internal/invoice"
)

func main() {
	outPath := "/tmp/invoice_theme_prototype.pdf"
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}

	logoPath, err := writePlaceholderLogo()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating placeholder logo:", err)
		os.Exit(1)
	}
	defer os.Remove(logoPath)

	eeg, member, inv, vat, energyRows, generationRows, history := invoice.SampleFixtureData()
	eeg.LogoPath = logoPath

	data, err := invoice.GeneratePDFThemed(inv, eeg, member, vat, history, energyRows, generationRows, invoice.DefaultOikosTheme())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error generating PDF:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing PDF:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", outPath)
}

// writePlaceholderLogo generates a small solid-color placeholder logo PNG at
// runtime (no real logo asset exists in the repo) and returns its temp path.
func writePlaceholderLogo() (string, error) {
	const w, h = 300, 120
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	accent := color.RGBA{R: 74, G: 93, B: 66, A: 255} // dark green, echoes the reference logo
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, accent)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	f, err := os.CreateTemp("", "pdfsample-logo-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(buf.Bytes()); err != nil {
		return "", err
	}
	return f.Name(), nil
}
