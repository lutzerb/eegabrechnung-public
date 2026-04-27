// @title           eegabrechnung API
// @version         1.0
// @description     Österreichische EEG (Energiegemeinschaft) Abrechnungsplattform — REST API
// @termsOfService  http://swagger.io/terms/

// @contact.name   eegabrechnung Support
// @contact.email  support@eeg.at

// @license.name  MIT

// @host      localhost:8101
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter: Bearer {token}

// @security BearerAuth

package main

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lutzerb/eegabrechnung/docs"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/billing"
	"github.com/lutzerb/eegabrechnung/internal/db"
	"github.com/lutzerb/eegabrechnung/internal/handler"
	apimiddleware "github.com/lutzerb/eegabrechnung/internal/middleware"
	"github.com/lutzerb/eegabrechnung/internal/repository"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	dsn := getEnv("DATABASE_URL", "postgres://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung?sslmode=disable")
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Error("JWT_SECRET environment variable is not set — refusing to start")
		os.Exit(1)
	}

	// Invoice / PDF configuration
	invoiceDir := getEnv("INVOICE_DIR", "/data/invoices")
	billingCfg := billing.Config{
		InvoiceDir: invoiceDir,
	}

	// Credential encryption key (AES-256, base64-encoded 32 bytes).
	encKey := mustDecodeEncKey(log, getEnv("CREDENTIAL_ENCRYPTION_KEY", ""))

	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Phase 3 backfill: fix prosumer invoices where split amounts don't match net_amount.
	// This is idempotent — exits immediately when no affected rows remain.
	if err := billing.BackfillSplitAmounts(ctx, pool); err != nil {
		log.Error("split amount backfill failed", "error", err)
		// Non-fatal: continue startup; EA import degrades gracefully for un-fixed rows.
	}

	// Repositories
	eegRepo := repository.NewEEGRepository(pool, encKey)
	memberRepo := repository.NewMemberRepository(pool)
	meterPointRepo := repository.NewMeterPointRepository(pool)
	readingRepo := repository.NewReadingRepository(pool)
	invoiceRepo := repository.NewInvoiceRepository(pool)
	billingRunRepo := repository.NewBillingRunRepository(pool)
	edaMessageRepo := repository.NewEDAMessageRepository(pool)
	edaProcessRepo := repository.NewEDAProcessRepository(pool)
	edaErrorRepo := repository.NewEDAErrorRepository(pool)
	workerStatusRepo := repository.NewEDAWorkerStatusRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	tariffRepo := repository.NewTariffRepository(pool)
	participationRepo := repository.NewParticipationRepository(pool)

	// Services
	billingSvc := billing.NewService(pool, eegRepo, memberRepo, readingRepo, invoiceRepo, billingRunRepo, tariffRepo, billingCfg)

	// Handlers
	eegHandler := handler.NewEEGHandler(eegRepo, memberRepo, meterPointRepo, participationRepo)
	importHandler := handler.NewImportHandler(eegRepo, memberRepo, meterPointRepo, readingRepo)
	billingHandler := handler.NewBillingHandler(billingSvc, invoiceRepo, billingRunRepo, memberRepo, eegRepo)
	memberHandler := handler.NewMemberHandler(memberRepo, meterPointRepo, eegRepo, edaProcessRepo, jobRepo, participationRepo)
	meterPointHandler := handler.NewMeterPointHandler(meterPointRepo, memberRepo)
	statsHandler := handler.NewStatsHandler(eegRepo, edaMessageRepo)
	authHandler := handler.NewAuthHandler(userRepo, jwtSecret)
	oemagHandler := handler.NewOemagHandler(eegRepo)
	sepaHandler := handler.NewSEPAHandler(eegRepo, memberRepo, invoiceRepo)
	reportHandler := handler.NewReportHandler(reportRepo, eegRepo)
	userHandler := handler.NewUserHandler(userRepo, eegRepo)
	tariffHandler := handler.NewTariffHandler(tariffRepo)
	edaWorkerURL := getEnv("EDA_WORKER_URL", "http://eda-worker:8081")
	edaHandler := handler.NewEDAHandler(eegRepo, meterPointRepo, edaProcessRepo, jobRepo, edaErrorRepo, workerStatusRepo, edaWorkerURL)
	participationHandler := handler.NewParticipationHandler(participationRepo, eegRepo)
	accountingHandler := handler.NewAccountingHandler(eegRepo, invoiceRepo, memberRepo)
	backupHandler := handler.NewBackupHandler(pool, eegRepo, memberRepo, meterPointRepo, readingRepo, invoiceRepo, billingRunRepo, tariffRepo, participationRepo)
	searchHandler := handler.NewSearchHandler(memberRepo, meterPointRepo, invoiceRepo, eegRepo)

	webBaseURL := getEnv("WEB_BASE_URL", "http://localhost:3001")
	onboardingRepo := repository.NewOnboardingRepository(pool)
	memberEmailRepo := repository.NewMemberEmailRepository(pool)
	eegDocumentRepo := repository.NewEEGDocumentRepository(pool)

	onboardingHandler := handler.NewOnboardingHandler(onboardingRepo, eegRepo, memberRepo, meterPointRepo, eegDocumentRepo, pool, webBaseURL)

	// Background: send 72-hour reminder emails for unconverted eda_sent requests.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			onboardingHandler.RunReminderCheck(ctx)
			<-ticker.C
		}
	}()

	portalRepo := repository.NewMemberPortalRepository(pool)
	portalHandler := handler.NewMemberPortalHandler(portalRepo, memberRepo, meterPointRepo, participationRepo, readingRepo, invoiceRepo, eegRepo, edaProcessRepo, jobRepo, webBaseURL)

	memberEmailHandler := handler.NewMemberEmailHandler(memberEmailRepo, memberRepo, eegRepo)
	eegDocumentHandler := handler.NewEEGDocumentHandler(eegDocumentRepo, portalRepo, eegRepo)

	eaRepo := repository.NewEARepository(pool)
	eaHandler := handler.NewEAHandler(eaRepo, invoiceDir)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(securityHeaders)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Rate limiters for public endpoints
	loginLimiter := apimiddleware.NewIPRateLimiter(10, time.Minute)         // 10 login attempts/min/IP
	onboardingLimiter := apimiddleware.NewIPRateLimiter(5, time.Minute)     // 5 submissions/min/IP
	portalLimiter := apimiddleware.NewIPRateLimiter(10, time.Minute)        // 10 portal requests/min/IP

	// Public auth endpoint
	r.With(loginLimiter.Middleware).Post("/api/v1/auth/login", authHandler.Login)

	// Public onboarding (no auth required)
	r.Get("/api/v1/public/eegs/{eegID}/info", onboardingHandler.GetPublicEEGInfo)
	r.With(onboardingLimiter.Middleware).Post("/api/v1/public/eegs/{eegID}/onboarding", onboardingHandler.SubmitOnboarding)
	r.Get("/api/v1/public/onboarding/status/{token}", onboardingHandler.GetOnboardingStatus)
	r.With(onboardingLimiter.Middleware).Post("/api/v1/public/onboarding/resend-token", onboardingHandler.ResendToken)
	r.With(onboardingLimiter.Middleware).Post("/api/v1/public/eegs/{eegID}/onboarding/verify-email", onboardingHandler.VerifyEmail)
	r.Post("/api/v1/public/eegs/{eegID}/onboarding/verify/{token}", onboardingHandler.ConfirmEmailVerification)

	// Public document download (no auth — for onboarding page AGB links)
	r.Get("/api/v1/public/eegs/{eegID}/documents/{docID}", eegDocumentHandler.PublicDownloadDocument)

	// Member portal (public, uses its own session token mechanism)
	r.With(portalLimiter.Middleware).Post("/api/v1/public/portal/request-link", portalHandler.RequestLink)
	r.With(portalLimiter.Middleware).Post("/api/v1/public/portal/exchange", portalHandler.ExchangeToken)
	r.Get("/api/v1/public/portal/me", portalHandler.GetMe)
	r.Get("/api/v1/public/portal/energy", portalHandler.GetEnergy)
	r.Get("/api/v1/public/portal/invoices", portalHandler.GetInvoices)
	r.Get("/api/v1/public/portal/invoices/{invoiceID}/pdf", portalHandler.GetInvoicePDF)
	r.Get("/api/v1/public/portal/documents", eegDocumentHandler.PortalListDocuments)
	r.Get("/api/v1/public/portal/documents/{docID}", eegDocumentHandler.PortalDownloadDocument)
	r.Get("/api/v1/public/portal/meter-points", portalHandler.GetMeterPoints)
	r.Post("/api/v1/public/portal/change-factor", portalHandler.ChangeParticipationFactor)

	authMiddleware := auth.Middleware(jwtSecret)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)

		// EEG endpoints
		r.Get("/eegs", eegHandler.ListEEGs)
		r.Post("/eegs", eegHandler.CreateEEG)
		r.Get("/eegs/{eegID}", eegHandler.GetEEG)
		r.Put("/eegs/{eegID}", eegHandler.UpdateEEG)
		r.Delete("/eegs/{eegID}", eegHandler.DeleteEEG)
		r.Get("/eegs/{eegID}/members", eegHandler.ListMembers)

		// Gap alert
		r.Get("/eegs/{eegID}/gap-alerts", eegHandler.ListGapAlerts)

		// Import / Export endpoints
		r.Post("/eegs/{eegID}/import/stammdaten", importHandler.ImportStammdaten)
		r.Post("/eegs/{eegID}/import/energiedaten", importHandler.ImportEnergieDaten)
		r.Post("/eegs/{eegID}/import/energiedaten/preview", importHandler.PreviewEnergieDaten)
		r.Get("/eegs/{eegID}/readings/coverage", importHandler.GetCoverage)
		r.Get("/eegs/{eegID}/export/stammdaten", eegHandler.ExportStammdaten)
		r.Get("/eegs/{eegID}/logo", eegHandler.GetLogo)
		r.Post("/eegs/{eegID}/logo", eegHandler.UploadLogo)

		// Billing endpoints
		r.Post("/eegs/{eegID}/billing/run", billingHandler.RunBilling)
		r.Get("/eegs/{eegID}/billing/runs", billingHandler.ListBillingRuns)
		r.Get("/eegs/{eegID}/billing/runs/{runID}/invoices", billingHandler.ListInvoicesByRun)
		r.Get("/eegs/{eegID}/billing/runs/{runID}/zip", billingHandler.ZipBillingRun)
		r.Get("/eegs/{eegID}/billing/runs/{runID}/export", billingHandler.ExportBillingRun)
		r.Post("/eegs/{eegID}/billing/runs/{runID}/send-all", billingHandler.SendAllInvoices)
		r.Post("/eegs/{eegID}/billing/runs/{runID}/finalize", billingHandler.FinalizeBillingRun)
		r.Delete("/eegs/{eegID}/billing/runs/{runID}", billingHandler.DeleteBillingRun)
		r.Post("/eegs/{eegID}/billing/runs/{runID}/cancel", billingHandler.CancelBillingRun)
		r.Get("/eegs/{eegID}/invoices", billingHandler.ListInvoices)
		r.Post("/eegs/{eegID}/invoices/send-all", billingHandler.SendAllInvoices)
		r.Get("/eegs/{eegID}/invoices/{invoiceID}/pdf", billingHandler.GetInvoicePDF)
		r.Patch("/eegs/{eegID}/invoices/{invoiceID}/status", billingHandler.UpdateInvoiceStatus)
		r.Patch("/eegs/{eegID}/invoices/{invoiceID}/sepa-return", billingHandler.SetSepaReturn)
		r.Post("/eegs/{eegID}/invoices/{invoiceID}/resend", billingHandler.ResendInvoice)

		// Member CRUD
		r.Post("/eegs/{eegID}/members", memberHandler.CreateMember)
		r.Get("/eegs/{eegID}/members/{memberID}", memberHandler.GetMember)
		r.Put("/eegs/{eegID}/members/{memberID}", memberHandler.UpdateMember)
		r.Delete("/eegs/{eegID}/members/{memberID}", memberHandler.DeleteMember)
		r.Post("/eegs/{eegID}/members/{memberID}/austritt", memberHandler.Austritt)
		r.Get("/eegs/{eegID}/members/{memberID}/sepa-mandat", memberHandler.DownloadSepaMandat)

		// Meter point CRUD
		r.Post("/eegs/{eegID}/members/{memberID}/meter-points", meterPointHandler.CreateMeterPoint)
		r.Get("/eegs/{eegID}/meter-points/{meterPointID}", meterPointHandler.GetMeterPoint)
		r.Put("/eegs/{eegID}/meter-points/{meterPointID}", meterPointHandler.UpdateMeterPoint)
		r.Delete("/eegs/{eegID}/meter-points/{meterPointID}", meterPointHandler.DeleteMeterPoint)

		// Stats + EDA messages
		r.Get("/eegs/{eegID}/stats", statsHandler.GetStats)
		r.Get("/eegs/{eegID}/eda/messages", statsHandler.GetEDAMessages)
		r.Get("/eegs/{eegID}/eda/messages/{msgID}/xml", statsHandler.GetEDAMessageXML)

		// EDA process management
		r.Get("/eegs/{eegID}/eda/processes", edaHandler.ListProcesses)
		r.Post("/eegs/{eegID}/eda/anmeldung", edaHandler.Anmeldung)
		r.Post("/eegs/{eegID}/eda/teilnahmefaktor", edaHandler.TeilnahmefaktorAendern)
		r.Post("/eegs/{eegID}/eda/zaehlerstandsgang", edaHandler.ZaehlerstandsgangAnfordern)
		r.Post("/eegs/{eegID}/eda/widerruf", edaHandler.WiderrufEEG)
		r.Post("/eegs/{eegID}/eda/podlist", edaHandler.PODList)
		r.Get("/eegs/{eegID}/eda/errors", edaHandler.ListErrors)
		r.Post("/eegs/{eegID}/eda/poll-now", edaHandler.PollNow)
		r.Get("/eda/worker-status", edaHandler.GetWorkerStatus)

		// SEPA payment files
		r.Get("/eegs/{eegID}/sepa/pain001", sepaHandler.DownloadPain001)
		r.Get("/eegs/{eegID}/sepa/pain008", sepaHandler.DownloadPain008)
		r.Post("/eegs/{eegID}/sepa/camt054", sepaHandler.ImportCAMT054)

		// Report endpoints
		r.Get("/eegs/{eegID}/reports/energy", reportHandler.GetMonthlyEnergy)
		r.Get("/eegs/{eegID}/reports/members", reportHandler.GetMemberStats)
		r.Get("/eegs/{eegID}/reports/annual", reportHandler.GetAnnualReport)
		r.Get("/eegs/{eegID}/energy/summary", reportHandler.GetEnergySummary)
		r.Get("/eegs/{eegID}/energy/members", reportHandler.GetRawMemberEnergy)

		// Search
		r.Get("/eegs/{eegID}/search", searchHandler.Search)

		// Backup / Restore
		r.Get("/eegs/{eegID}/backup", backupHandler.Export)
		r.Post("/eegs/{eegID}/restore", backupHandler.Restore)

		// Accounting / DATEV export
		r.Get("/eegs/{eegID}/accounting/export", accountingHandler.Export)

		// Tariff schedules
		r.Get("/eegs/{eegID}/tariffs", tariffHandler.ListSchedules)
		r.Post("/eegs/{eegID}/tariffs", tariffHandler.CreateSchedule)
		r.Get("/eegs/{eegID}/tariffs/{scheduleID}", tariffHandler.GetSchedule)
		r.Put("/eegs/{eegID}/tariffs/{scheduleID}", tariffHandler.UpdateSchedule)
		r.Delete("/eegs/{eegID}/tariffs/{scheduleID}", tariffHandler.DeleteSchedule)
		r.Put("/eegs/{eegID}/tariffs/{scheduleID}/entries", tariffHandler.SetEntries)
		r.Post("/eegs/{eegID}/tariffs/{scheduleID}/activate", tariffHandler.ActivateSchedule)
		r.Delete("/eegs/{eegID}/tariffs/{scheduleID}/activate", tariffHandler.DeactivateSchedule)

		// Onboarding (approve triggers full member creation + EDA)
		r.Get("/eegs/{eegID}/onboarding", onboardingHandler.ListOnboarding)
		r.Get("/eegs/{eegID}/onboarding/{id}", onboardingHandler.GetOnboardingByID)
		r.Patch("/eegs/{eegID}/onboarding/{id}", onboardingHandler.UpdateOnboardingStatus)
		r.Delete("/eegs/{eegID}/onboarding/{id}", onboardingHandler.DeleteOnboarding)

		// Bulk email campaigns
		r.Get("/eegs/{eegID}/communications", memberEmailHandler.ListCampaigns)
		r.Get("/eegs/{eegID}/communications/{id}", memberEmailHandler.GetCampaign)
		r.Post("/eegs/{eegID}/communications", memberEmailHandler.SendCampaign)

		// EEG documents
		r.Get("/eegs/{eegID}/documents", eegDocumentHandler.ListDocuments)
		r.Post("/eegs/{eegID}/documents", eegDocumentHandler.UploadDocument)
		r.Delete("/eegs/{eegID}/documents/{docID}", eegDocumentHandler.DeleteDocument)
		r.Patch("/eegs/{eegID}/documents/{docID}", eegDocumentHandler.PatchDocument)
		r.Get("/eegs/{eegID}/documents/{docID}/download", eegDocumentHandler.DownloadDocument)

		// Mehrfachteilnahme — meter point participation records
		r.Get("/eegs/{eegID}/participations", participationHandler.ListByEEG)
		r.Post("/eegs/{eegID}/participations", participationHandler.Create)
		r.Put("/eegs/{eegID}/participations/{id}", participationHandler.Update)
		r.Delete("/eegs/{eegID}/participations/{id}", participationHandler.Delete)

		// Admin: user management (admin role required)
		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.RequireAdmin)
			r.Get("/users", userHandler.ListUsers)
			r.Post("/users", userHandler.CreateUser)
			r.Get("/users/{userID}", userHandler.GetUser)
			r.Put("/users/{userID}", userHandler.UpdateUser)
			r.Delete("/users/{userID}", userHandler.DeleteUser)
			r.Get("/users/{userID}/eegs", userHandler.GetUserEEGs)
			r.Put("/users/{userID}/eegs", userHandler.SetUserEEGs)
		})

		// E/A-Buchhaltung
		r.Get("/eegs/{eegID}/ea/settings", eaHandler.GetSettings)
		r.Put("/eegs/{eegID}/ea/settings", eaHandler.UpdateSettings)
		r.Get("/eegs/{eegID}/ea/konten", eaHandler.ListKonten)
		r.Post("/eegs/{eegID}/ea/konten", eaHandler.CreateKonto)
		r.Put("/eegs/{eegID}/ea/konten/{kontoID}", eaHandler.UpdateKonto)
		r.Delete("/eegs/{eegID}/ea/konten/{kontoID}", eaHandler.DeleteKonto)
		r.Get("/eegs/{eegID}/ea/buchungen", eaHandler.ListBuchungen)
		r.Post("/eegs/{eegID}/ea/buchungen", eaHandler.CreateBuchung)
		r.Get("/eegs/{eegID}/ea/buchungen/{buchungID}", eaHandler.GetBuchung)
		r.Put("/eegs/{eegID}/ea/buchungen/{buchungID}", eaHandler.UpdateBuchung)
		r.Delete("/eegs/{eegID}/ea/buchungen/{buchungID}", eaHandler.DeleteBuchung)
		r.Get("/eegs/{eegID}/ea/buchungen/{buchungID}/changelog", eaHandler.GetBuchungChangelog)
		r.Get("/eegs/{eegID}/ea/changelog", eaHandler.ListChangelog)
		r.Post("/eegs/{eegID}/ea/belege", eaHandler.UploadBeleg)
		r.Get("/eegs/{eegID}/ea/belege/{belegID}", eaHandler.GetBeleg)
		r.Delete("/eegs/{eegID}/ea/belege/{belegID}", eaHandler.DeleteBeleg)
		r.Get("/eegs/{eegID}/ea/saldenliste", eaHandler.GetSaldenliste)
		r.Get("/eegs/{eegID}/ea/kontenblatt/{kontoID}", eaHandler.GetKontenblatt)
		r.Get("/eegs/{eegID}/ea/jahresabschluss", eaHandler.GetJahresabschluss)
		r.Get("/eegs/{eegID}/ea/buchungen/export", eaHandler.ExportBuchungenXLSX)
		r.Get("/eegs/{eegID}/ea/uva", eaHandler.ListUVA)
		r.Post("/eegs/{eegID}/ea/uva", eaHandler.CreateUVAPeriode)
		r.Get("/eegs/{eegID}/ea/uva/{uvaID}/kennzahlen", eaHandler.GetUVAKennzahlen)
		r.Patch("/eegs/{eegID}/ea/uva/{uvaID}/eingereicht", eaHandler.SetUVAEingereicht)
		r.Get("/eegs/{eegID}/ea/uva/{uvaID}/export", eaHandler.ExportUVAXML)
		r.Get("/eegs/{eegID}/ea/erklaerungen/u1", eaHandler.GetU1)
		r.Get("/eegs/{eegID}/ea/erklaerungen/k1", eaHandler.GetK1)
		r.Get("/eegs/{eegID}/ea/erklaerungen/k2", eaHandler.GetK2)
		r.Get("/eegs/{eegID}/ea/import/preview", eaHandler.ImportPreview)
		r.Post("/eegs/{eegID}/ea/import/rechnungen", eaHandler.ImportRechnungen)
		r.Post("/eegs/{eegID}/ea/bank/import", eaHandler.ImportBank)
		r.Get("/eegs/{eegID}/ea/bank/transaktionen", eaHandler.ListBankTransaktionen)
		r.Post("/eegs/{eegID}/ea/bank/match", eaHandler.BestaetigeMatch)
		r.Delete("/eegs/{eegID}/ea/bank/transaktionen/{transaktionID}", eaHandler.IgnoriereBankTransaktion)

		// OeMAG market price
		r.Get("/oemag/marktpreis", oemagHandler.GetMarktpreis)
		r.Post("/oemag/refresh", oemagHandler.RefreshMarktpreis)
		r.Post("/eegs/{eegID}/oemag/sync", oemagHandler.SyncEEGPrice)
	})

	// Start auto-billing scheduler in background
	billingScheduler := billing.NewScheduler(eegRepo, readingRepo, billingSvc)
	go billingScheduler.Run(ctx)

	// Start gap-alert checker in background (hourly)
	gapChecker := billing.NewGapChecker(eegRepo, meterPointRepo)
	go gapChecker.Run(ctx)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // generous for PDF generation
		IdleTimeout:  120 * time.Second,
	}
	log.Info("starting server", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustDecodeEncKey(log *slog.Logger, b64str string) []byte {
	if b64str == "" {
		log.Error("CREDENTIAL_ENCRYPTION_KEY is not set — refusing to start")
		os.Exit(1)
	}
	key, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil || len(key) != 32 {
		log.Error("CREDENTIAL_ENCRYPTION_KEY must be base64-encoded exactly 32 bytes (AES-256)")
		os.Exit(1)
	}
	return key
}

// securityHeaders adds defensive HTTP headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// HSTS is set by the TLS terminator (Caddy/Cloudflare); omit here so HTTP dev mode isn't broken.
		next.ServeHTTP(w, r)
	})
}
