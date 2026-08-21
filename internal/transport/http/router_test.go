package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/application"
	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/platform/clock"
	"github.com/wyw14/cry-055/internal/platform/localnotify"
	"github.com/wyw14/cry-055/internal/repository/memory"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) (http.Handler, *memory.Store, domain.Instrument) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	lab, _ := domain.NewLaboratory("LAB-A", "Analytical", "A-101", "manager", now)
	if err := store.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	instrument, _ := domain.NewInstrument(domain.InstrumentInput{LaboratoryID: lab.ID, AssetNumber: "LAB-001", Name: "Gauge", Model: "PG-10", OwnerID: "owner", Criticality: domain.CriticalityCritical}, now)
	instrument.NextDueAt = now.Add(-time.Hour)
	instrument.Status = domain.StatusQualified
	if err := store.CreateInstrument(ctx, instrument); err != nil {
		t.Fatal(err)
	}
	fixed := clock.Fixed{Value: now}
	catalog := application.NewCatalogService(store, store, fixed)
	instruments := application.NewInstrumentService(store, store, store, fixed)
	plans := application.NewPlanService(store, store, fixed)
	executions := application.NewExecutionService(store, store, store, store, store, fixed)
	nonconformance := application.NewNonconformanceService(store, store, store, store, fixed)
	usage := application.NewUsageService(store, store, fixed, 30)
	alerts := application.NewAlertService(store, store, store, localnotify.New(), fixed)
	reports := application.NewReportService(store, store, store, store)
	router := NewRouter(Services{Catalog: catalog, Instruments: instruments, Plans: plans, Executions: executions, Nonconformance: nonconformance, Usage: usage, Alerts: alerts, Reports: reports}, Options{Logger: zap.NewNop(), RequestTimeout: time.Second})
	return router, store, instrument
}

func TestUsageCheckHTTPBlocksOverdueCriticalInstrument(t *testing.T) {
	router, store, instrument := testRouter(t)
	body, _ := json.Marshal(map[string]any{"batch_number": "BATCH-77", "operator_id": "operator"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/instruments/"+string(instrument.ID)+"/usage-checks", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Actor-ID", "operator")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusLocked {
		t.Fatalf("expected 423, got %d: %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["code"] != "INSTRUMENT_BLOCKED" {
		t.Fatalf("unexpected code: %v", payload["code"])
	}
	checks, err := store.ListUsageChecks(context.Background(), instrument.ID)
	if err != nil || len(checks) != 1 {
		t.Fatalf("expected persisted check, got %v %v", checks, err)
	}
}

func TestProtectedEndpointRequiresActor(t *testing.T) {
	router, _, _ := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instruments?page=1&size=20&sort=created_at", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing request id")
	}
}

func TestInstrumentListRejectsUnknownFilter(t *testing.T) {
	router, _, _ := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instruments?page=1&size=20&sort=created_at&filter%5Bsecret%5D=value", nil)
	request.Header.Set("X-Actor-ID", "operator")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}
