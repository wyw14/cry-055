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
	executions := application.NewExecutionService(store, store, store, store, fixed)
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

func TestInstrumentListFiltersEffectiveStatusBeforePaginationAndReturnsStableErrors(t *testing.T) {
	router, store, first := testRouter(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	second, err := domain.NewInstrument(domain.InstrumentInput{
		LaboratoryID: first.LaboratoryID,
		AssetNumber:  "LAB-002",
		Name:         "Temperature probe",
		Model:        "TP-20",
		OwnerID:      "owner",
		Criticality:  domain.CriticalityMajor,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	second.Status = domain.StatusQualified
	second.NextDueAt = now.Add(-2 * time.Hour)
	if err := store.CreateInstrument(ctx, second); err != nil {
		t.Fatal(err)
	}

	t.Run("effective status is filtered before pagination", func(t *testing.T) {
		listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instruments?page=2&size=1&sort=asset_number&order=asc&filter%5Bstatus%5D=overdue", nil)
		listRequest.Header.Set("X-Actor-ID", "operator")
		listResponse := httptest.NewRecorder()
		router.ServeHTTP(listResponse, listRequest)
		if listResponse.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", listResponse.Code, listResponse.Body.String())
		}
		var page domain.Page[domain.Instrument]
		if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Page != 2 || page.Size != 1 || page.Total != 2 {
			t.Fatalf("unexpected page metadata: page=%d size=%d total=%d", page.Page, page.Size, page.Total)
		}
		if len(page.Items) != 1 || page.Items[0].AssetNumber != "LAB-002" || page.Items[0].Status != domain.StatusOverdue {
			t.Fatalf("expected second overdue instrument, got %+v", page.Items)
		}
	})

	t.Run("invalid pagination has a stable error contract", func(t *testing.T) {
		errorRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instruments?page=not-a-number&size=1&sort=asset_number&filter%5Bstatus%5D=overdue", nil)
		errorRequest.Header.Set("X-Actor-ID", "operator")
		errorRequest.Header.Set("X-Request-ID", "list-contract-009")
		errorResponseRecorder := httptest.NewRecorder()
		router.ServeHTTP(errorResponseRecorder, errorRequest)
		if errorResponseRecorder.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %s", errorResponseRecorder.Code, errorResponseRecorder.Body.String())
		}
		var problem errorResponse
		if err := json.Unmarshal(errorResponseRecorder.Body.Bytes(), &problem); err != nil {
			t.Fatal(err)
		}
		if problem.Code != "VALIDATION_ERROR" || problem.Message != "request validation failed" {
			t.Fatalf("unexpected stable error: %+v", problem)
		}
		if problem.RequestID != "list-contract-009" || errorResponseRecorder.Header().Get("X-Request-ID") != problem.RequestID {
			t.Fatalf("request id mismatch: header=%q body=%q", errorResponseRecorder.Header().Get("X-Request-ID"), problem.RequestID)
		}
		if len(problem.FieldErrors) != 1 || problem.FieldErrors[0].Field != "page" {
			t.Fatalf("expected page field error, got %+v", problem.FieldErrors)
		}
	})

	t.Run("listing is read only", func(t *testing.T) {
		for _, instrumentID := range []domain.ID{first.ID, second.ID} {
			audits, err := store.ListAudit(ctx, "instrument", instrumentID)
			if err != nil || len(audits) != 0 {
				t.Fatalf("list must not append audits for %s: audits=%v err=%v", instrumentID, audits, err)
			}
		}
	})
}
