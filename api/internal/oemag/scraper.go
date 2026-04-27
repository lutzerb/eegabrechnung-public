// Package oemag fetches the monthly market prices published by OeMAG
// (Abwicklungsstelle für Ökostrom AG) at https://www.oem-ag.at/marktpreis.
package oemag

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const sourceURL = "https://www.oem-ag.at/marktpreis"

// MonthPrice holds the PV and wind market prices for a single month.
type MonthPrice struct {
	Month     int     `json:"month"`      // 1–12
	PVPrice   float64 `json:"pv_price"`   // ct/kWh, Photovoltaik & andere
	WindPrice float64 `json:"wind_price"` // ct/kWh, Windkraft
}

// YearPrices holds all published monthly prices for one calendar year.
type YearPrices struct {
	Year   int          `json:"year"`
	Prices []MonthPrice `json:"prices"`
	Static bool         `json:"static"` // true = hardcoded historical data
}

// MarktpreisResult is the full response returned by the API.
type MarktpreisResult struct {
	Years     []YearPrices `json:"years"`
	ScrapedAt time.Time    `json:"scraped_at"`
}

// NewMonth describes a month that was not in the cache on the previous scrape.
type NewMonth struct {
	Year  int        `json:"year"`
	Price MonthPrice `json:"price"`
}

// RefreshResult is returned by Refresh().
type RefreshResult struct {
	All       []YearPrices `json:"all"`
	NewMonths []NewMonth   `json:"new_months"`
	ScrapedAt time.Time    `json:"scraped_at"`
}

// ── Static historical data ────────────────────────────────────────────────────
// Source: OeMAG Marktpreise 2025 PDF + Berechnungsgrundlagen Excel.
// In 2025 there were no Ausgleichsenergie deductions (§ 13 Abs. 3 ÖSG 2012),
// so PV price == Wind price for every month.

var static2025 = YearPrices{
	Year:   2025,
	Static: true,
	Prices: []MonthPrice{
		{Month: 1, PVPrice: 9.730, WindPrice: 9.730},
		{Month: 2, PVPrice: 9.730, WindPrice: 9.730},
		{Month: 3, PVPrice: 6.007, WindPrice: 6.007},
		{Month: 4, PVPrice: 5.855, WindPrice: 5.855},
		{Month: 5, PVPrice: 5.855, WindPrice: 5.855},
		{Month: 6, PVPrice: 5.855, WindPrice: 5.855},
		{Month: 7, PVPrice: 5.965, WindPrice: 5.965},
		{Month: 8, PVPrice: 5.892, WindPrice: 5.892},
		{Month: 9, PVPrice: 5.892, WindPrice: 5.892},
		{Month: 10, PVPrice: 9.008, WindPrice: 9.008},
		{Month: 11, PVPrice: 9.167, WindPrice: 9.167},
		{Month: 12, PVPrice: 9.167, WindPrice: 9.167},
	},
}

// ── In-memory cache for new-month detection ──────────────────────────────────

var (
	cacheMu    sync.Mutex
	knownKeys  = map[string]bool{} // "YYYY-MM" → true
	cacheReady bool
)

func cacheKey(year, month int) string {
	return fmt.Sprintf("%d-%02d", year, month)
}

// updateCache updates the internal cache and returns newly discovered months.
// On the very first call (empty cache) new_months is always empty so that the
// initial page load does not flood the user with "new months" banners.
func updateCache(years []YearPrices) []NewMonth {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	var newMonths []NewMonth
	for _, yp := range years {
		for _, p := range yp.Prices {
			k := cacheKey(yp.Year, p.Month)
			if cacheReady && !knownKeys[k] {
				newMonths = append(newMonths, NewMonth{Year: yp.Year, Price: p})
			}
			knownKeys[k] = true
		}
	}
	cacheReady = true
	return newMonths
}

// ── Public API ────────────────────────────────────────────────────────────────

// FetchAll returns static historical data combined with a fresh scrape of the
// current year's prices.
func FetchAll() (*MarktpreisResult, error) {
	currentYear, err := scrapeCurrentYear()
	if err != nil {
		return nil, err
	}

	years := buildYears(currentYear)
	return &MarktpreisResult{
		Years:     years,
		ScrapedAt: time.Now(),
	}, nil
}

// Refresh re-scrapes the OeMAG page, updates the cache, and reports any
// months that were not present on the previous call.
func Refresh() (*RefreshResult, error) {
	currentYear, err := scrapeCurrentYear()
	if err != nil {
		return nil, err
	}

	years := buildYears(currentYear)
	newMonths := updateCache(years)

	return &RefreshResult{
		All:       years,
		NewMonths: newMonths,
		ScrapedAt: time.Now(),
	}, nil
}

// FetchCurrentMonth returns the most recently published monthly price.
func FetchCurrentMonth() (*MonthPrice, error) {
	result, err := FetchAll()
	if err != nil {
		return nil, err
	}

	currentMonth := int(time.Now().Month())
	currentYear := time.Now().Year()

	// Look in current year first
	for _, yp := range result.Years {
		if yp.Year == currentYear {
			for _, p := range yp.Prices {
				if p.Month == currentMonth {
					return &p, nil
				}
			}
			// Current month not yet published — use latest in current year
			if len(yp.Prices) > 0 {
				latest := yp.Prices[len(yp.Prices)-1]
				return &latest, nil
			}
		}
	}

	// Fall back to last month of the most recent year
	for i := len(result.Years) - 1; i >= 0; i-- {
		if len(result.Years[i].Prices) > 0 {
			latest := result.Years[i].Prices[len(result.Years[i].Prices)-1]
			return &latest, nil
		}
	}
	return nil, fmt.Errorf("keine Preise verfügbar")
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// buildYears assembles the final slice: static years + current scraped year.
// Static years are prepended; the scraped current year is appended last.
func buildYears(currentYear YearPrices) []YearPrices {
	var years []YearPrices

	// Static historical years (add more here as they become available)
	if currentYear.Year != static2025.Year {
		years = append(years, static2025)
	}

	years = append(years, currentYear)
	return years
}

// scrapeCurrentYear scrapes the OeMAG website and returns prices for the
// current calendar year.
func scrapeCurrentYear() (YearPrices, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(sourceURL)
	if err != nil {
		return YearPrices{}, fmt.Errorf("fetch oemag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return YearPrices{}, fmt.Errorf("oemag returned HTTP %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return YearPrices{}, fmt.Errorf("parse html: %w", err)
	}

	prices, err := parseTable(doc)
	if err != nil {
		return YearPrices{}, err
	}

	return YearPrices{
		Year:   time.Now().Year(),
		Prices: prices,
		Static: false,
	}, nil
}

// ── HTML parsing ─────────────────────────────────────────────────────────────

var germanMonths = map[string]int{
	"jänner":    1,
	"januar":    1,
	"februar":   2,
	"märz":      3,
	"maerz":     3,
	"april":     4,
	"mai":       5,
	"juni":      6,
	"juli":      7,
	"august":    8,
	"september": 9,
	"oktober":   10,
	"november":  11,
	"dezember":  12,
}

func parseMonthName(s string) (int, bool) {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.TrimSpace(s)
	m, ok := germanMonths[s]
	return m, ok
}

// parsePrice converts "8,842 ct/kWh" → 8.842.
func parsePrice(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, " ct/kWh", "", 1)
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

func findTable(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "table" {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, "table-bordered") {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findTable(c); found != nil {
			return found
		}
	}
	return nil
}

func parseTable(doc *html.Node) ([]MonthPrice, error) {
	table := findTable(doc)
	if table == nil {
		return nil, fmt.Errorf("Preistabelle nicht auf der OeMAG-Seite gefunden")
	}

	var tbody *html.Node
	for c := table.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tbody" {
			tbody = c
			break
		}
	}
	if tbody == nil {
		return nil, fmt.Errorf("tbody nicht in der Preistabelle gefunden")
	}

	var prices []MonthPrice
	for row := tbody.FirstChild; row != nil; row = row.NextSibling {
		if row.Type != html.ElementNode || row.Data != "tr" {
			continue
		}
		var cells []*html.Node
		for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
			if cell.Type == html.ElementNode && cell.Data == "td" {
				cells = append(cells, cell)
			}
		}
		if len(cells) < 3 {
			continue
		}
		month, ok := parseMonthName(textContent(cells[0]))
		if !ok {
			continue
		}
		pvPrice, err := parsePrice(textContent(cells[1]))
		if err != nil {
			continue
		}
		windPrice, err := parsePrice(textContent(cells[2]))
		if err != nil {
			continue
		}
		prices = append(prices, MonthPrice{
			Month:     month,
			PVPrice:   pvPrice,
			WindPrice: windPrice,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("keine Preiszeilen in der Tabelle gefunden")
	}
	return prices, nil
}
