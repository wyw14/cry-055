package httptransport

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-055/internal/application"
	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/middleware"
)

type handlers struct{ services Services }

func newHandlers(services Services) *handlers { return &handlers{services: services} }

type laboratoryRequest struct {
	Code      string `json:"code" validate:"required,max=32"`
	Name      string `json:"name" validate:"required,max=160"`
	Location  string `json:"location" validate:"max=200"`
	ManagerID string `json:"manager_id" validate:"required"`
}

func (h *handlers) createLaboratory(c *gin.Context) {
	var request laboratoryRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Catalog.CreateLaboratory(c.Request.Context(), request.Code, request.Name, request.Location, domain.ID(request.ManagerID))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}

type itemRequest struct {
	Code                string   `json:"code" validate:"required,max=32"`
	Name                string   `json:"name" validate:"required,max=160"`
	PeriodDays          int      `json:"period_days" validate:"required,min=1"`
	WarningDays         int      `json:"warning_days" validate:"min=0"`
	Tolerance           float64  `json:"tolerance" validate:"required,gt=0"`
	Unit                string   `json:"unit" validate:"required,max=24"`
	ReferenceStandardID string   `json:"reference_standard_id" validate:"required"`
	ApplicableModels    []string `json:"applicable_models" validate:"max=100,dive,max=120"`
}

func (h *handlers) createCalibrationItem(c *gin.Context) {
	var request itemRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Catalog.CreateCalibrationItem(c.Request.Context(), application.CalibrationItemInput{Code: request.Code, Name: request.Name, PeriodDays: request.PeriodDays, WarningDays: request.WarningDays, Tolerance: request.Tolerance, Unit: request.Unit, ReferenceStandardID: domain.ID(request.ReferenceStandardID), ApplicableModels: request.ApplicableModels})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}

type instrumentRequest struct {
	LaboratoryID string             `json:"laboratory_id" validate:"required"`
	AssetNumber  string             `json:"asset_number" validate:"required,max=64"`
	Name         string             `json:"name" validate:"required,max=160"`
	Model        string             `json:"model" validate:"max=120"`
	SerialNumber string             `json:"serial_number" validate:"max=120"`
	Location     string             `json:"location" validate:"max=200"`
	OwnerID      string             `json:"owner_id" validate:"required"`
	Criticality  domain.Criticality `json:"criticality" validate:"required"`
}

func (h *handlers) registerInstrument(c *gin.Context) {
	var request instrumentRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Instruments.Register(c.Request.Context(), domain.InstrumentInput{LaboratoryID: domain.ID(request.LaboratoryID), AssetNumber: request.AssetNumber, Name: request.Name, Model: request.Model, SerialNumber: request.SerialNumber, Location: request.Location, OwnerID: domain.ID(request.OwnerID), Criticality: request.Criticality})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}
func (h *handlers) listInstruments(c *gin.Context) {
	request, err := pageRequest(c)
	if err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Instruments.List(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h *handlers) getInstrument(c *gin.Context) {
	id, err := requireParam(c, "instrumentID")
	if err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Instruments.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("ETag", `"`+fmt.Sprint(value.Version)+`"`)
	c.JSON(http.StatusOK, value)
}

type scheduleRequest struct {
	NextDueAt time.Time `json:"next_due_at" validate:"required"`
}

func (h *handlers) scheduleInstrument(c *gin.Context) {
	id, err := requireParam(c, "instrumentID")
	if err != nil {
		writeError(c, err)
		return
	}
	version, err := parseVersion(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var request scheduleRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Instruments.Schedule(c.Request.Context(), id, request.NextDueAt, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

type usageRequest struct {
	BatchNumber string `json:"batch_number" validate:"required,max=80"`
	OperatorID  string `json:"operator_id" validate:"required"`
}

func (h *handlers) checkUsage(c *gin.Context) {
	id, err := requireParam(c, "instrumentID")
	if err != nil {
		writeError(c, err)
		return
	}
	var request usageRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Usage.Check(c.Request.Context(), id, request.BatchNumber, domain.ID(request.OperatorID))
	if err != nil {
		if errors.Is(err, domain.ErrInstrumentBlocked) {
			c.JSON(http.StatusLocked, gin.H{"code": "INSTRUMENT_BLOCKED", "message": "instrument is not available", "usage_check": value, "request_id": middleware.GetRequestID(c)})
			return
		}
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h *handlers) instrumentTrace(c *gin.Context) {
	id, err := requireParam(c, "instrumentID")
	if err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Reports.InstrumentTrace(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

type planRequest struct {
	InstrumentID    string    `json:"instrument_id" validate:"required"`
	ItemID          string    `json:"item_id" validate:"required"`
	AssignedTo      string    `json:"assigned_to" validate:"required"`
	LastQualifiedAt time.Time `json:"last_qualified_at" validate:"required"`
}

func (h *handlers) createPlan(c *gin.Context) {
	var request planRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Plans.Generate(c.Request.Context(), domain.ID(request.InstrumentID), domain.ID(request.ItemID), domain.ID(request.AssignedTo), request.LastQualifiedAt)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}

type rescheduleRequest struct {
	DueAt  time.Time `json:"due_at" validate:"required"`
	Reason string    `json:"reason" validate:"required,max=500"`
}

func (h *handlers) reschedulePlan(c *gin.Context) {
	id, err := requireParam(c, "planID")
	if err != nil {
		writeError(c, err)
		return
	}
	version, err := parseVersion(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var request rescheduleRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Plans.Reschedule(c.Request.Context(), id, request.DueAt, request.Reason, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

type executionRequest struct {
	InstrumentID  string               `json:"instrument_id" validate:"required"`
	ItemID        string               `json:"item_id" validate:"required"`
	PlanID        string               `json:"plan_id" validate:"required"`
	ExecutorID    string               `json:"executor_id" validate:"required"`
	Measurements  []domain.Measurement `json:"measurements" validate:"required,min=1,dive"`
	CertificateID string               `json:"certificate_id"`
	CompletedAt   time.Time            `json:"completed_at" validate:"required"`
}

func (h *handlers) recordExecution(c *gin.Context) {
	var request executionRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Executions.Record(c.Request.Context(), application.ExecutionInput{InstrumentID: domain.ID(request.InstrumentID), ItemID: domain.ID(request.ItemID), PlanID: domain.ID(request.PlanID), ExecutorID: domain.ID(request.ExecutorID), Measurements: request.Measurements, CertificateID: domain.ID(request.CertificateID), CompletedAt: request.CompletedAt})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}

type reviewRequest struct {
	ReviewerID string `json:"reviewer_id" validate:"required"`
	Comment    string `json:"comment" validate:"max=500"`
}

func (h *handlers) reviewExecution(c *gin.Context) {
	id, err := requireParam(c, "executionID")
	if err != nil {
		writeError(c, err)
		return
	}
	var request reviewRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Executions.Review(c.Request.Context(), id, domain.ID(request.ReviewerID), request.Comment)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

type impactRequest struct {
	Batches []domain.ImpactedBatch `json:"batches" validate:"required,min=1,dive"`
}

func (h *handlers) assessImpact(c *gin.Context) {
	id, err := requireParam(c, "nonconformanceID")
	if err != nil {
		writeError(c, err)
		return
	}
	version, err := parseVersion(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var request impactRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Nonconformance.AssessImpact(c.Request.Context(), id, request.Batches, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h *handlers) requestRetest(c *gin.Context) {
	id, err := requireParam(c, "nonconformanceID")
	if err != nil {
		writeError(c, err)
		return
	}
	version, err := parseVersion(c)
	if err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Nonconformance.RequestRetest(c.Request.Context(), id, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

type restoreRequest struct {
	Comment string `json:"comment" validate:"required,max=500"`
}

func (h *handlers) restoreInstrument(c *gin.Context) {
	id, err := requireParam(c, "nonconformanceID")
	if err != nil {
		writeError(c, err)
		return
	}
	version, err := parseVersion(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var request restoreRequest
	if err := bindJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	actor := application.MetadataFromContext(c.Request.Context()).Actor
	value, err := h.services.Nonconformance.Restore(c.Request.Context(), id, actor, request.Comment, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h *handlers) listAlerts(c *gin.Context) {
	request, err := pageRequest(c)
	if err != nil {
		writeError(c, err)
		return
	}
	value, err := h.services.Alerts.List(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
