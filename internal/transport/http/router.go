package httptransport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-055/internal/application"
	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/middleware"
	"github.com/wyw14/cry-055/internal/platform/buildinfo"
	"go.uber.org/zap"
)

type Services struct {
	Catalog        *application.CatalogService
	Instruments    *application.InstrumentService
	Plans          *application.PlanService
	Executions     *application.ExecutionService
	Nonconformance *application.NonconformanceService
	Usage          *application.UsageService
	Alerts         *application.AlertService
	Reports        *application.ReportService
}
type Options struct {
	Logger         *zap.Logger
	RequestTimeout time.Duration
	AllowedOrigin  string
	Ready          func() error
}

func NewRouter(services Services, options Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.SecurityHeaders(), middleware.CORS(options.AllowedOrigin), middleware.Recovery(options.Logger), middleware.Timeout(options.RequestTimeout))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "source": buildinfo.CurrentSource()})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if options.Ready != nil {
			if err := options.Ready(); err != nil {
				c.JSON(http.StatusServiceUnavailable, errorResponse{Code: "NOT_READY", Message: "database is unavailable", FieldErrors: []domain.FieldError{}, RequestID: middleware.GetRequestID(c)})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	api := router.Group("/api/v1")
	api.Use(middleware.LocalAuth())
	handlers := newHandlers(services)
	api.POST("/laboratories", middleware.RequireRole("quality_manager"), handlers.createLaboratory)
	api.POST("/calibration-items", middleware.RequireRole("quality_manager"), handlers.createCalibrationItem)
	api.POST("/instruments", handlers.registerInstrument)
	api.GET("/instruments", handlers.listInstruments)
	api.GET("/instruments/:instrumentID", handlers.getInstrument)
	api.PUT("/instruments/:instrumentID/schedule", handlers.scheduleInstrument)
	api.POST("/instruments/:instrumentID/usage-checks", handlers.checkUsage)
	api.GET("/instruments/:instrumentID/trace", handlers.instrumentTrace)
	api.POST("/plans", handlers.createPlan)
	api.PUT("/plans/:planID/reschedule", handlers.reschedulePlan)
	api.POST("/executions", handlers.recordExecution)
	api.POST("/executions/:executionID/review", handlers.reviewExecution)
	api.PUT("/nonconformances/:nonconformanceID/impact", handlers.assessImpact)
	api.POST("/nonconformances/:nonconformanceID/retest", handlers.requestRetest)
	api.POST("/nonconformances/:nonconformanceID/restore", middleware.RequireRole("quality_manager"), handlers.restoreInstrument)
	api.GET("/alerts", handlers.listAlerts)
	return router
}
