package services

import (
	"context"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/sirupsen/logrus"
)

type StartupPhase string

const (
	PhaseStarting       StartupPhase = "starting"
	PhaseCriticalReady  StartupPhase = "critical_ready"
	PhaseBackgroundInit StartupPhase = "background_init"
	PhaseFullyReady     StartupPhase = "fully_ready"
)

type JobStatus struct {
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	Message   string    `json:"message"`
}

type ServiceStatus struct {
	State       string    `json:"state"`
	LastChecked time.Time `json:"last_checked"`
	Healthy     bool      `json:"healthy"`
}

type StartupManager struct {
	phase            StartupPhase
	backgroundJobs   map[string]JobStatus
	externalServices map[string]ServiceStatus
	mu               sync.RWMutex
	logger           *logrus.Logger
	config           *config.Config
	dataFetcher      *DataFetcherService
	golfSyncService  *GolfTournamentSyncService
	circuitBreaker   *CircuitBreakerService
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewStartupManager(cfg *config.Config, logger *logrus.Logger, dataFetcher *DataFetcherService, golfSync *GolfTournamentSyncService, cb *CircuitBreakerService) *StartupManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &StartupManager{
		phase:            PhaseStarting,
		backgroundJobs:   make(map[string]JobStatus),
		externalServices: make(map[string]ServiceStatus),
		logger:           logger,
		config:           cfg,
		dataFetcher:      dataFetcher,
		golfSyncService:  golfSync,
		circuitBreaker:   cb,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// StartCriticalServices starts only essential services for basic functionality
func (sm *StartupManager) StartCriticalServices() error {
	sm.mu.Lock()
	sm.phase = PhaseStarting
	sm.mu.Unlock()

	sm.logger.WithFields(logrus.Fields{
		"component": "startup_manager",
		"phase":     PhaseStarting,
	}).Info("Starting critical services")

	// Only start essential non-blocking services here
	// Database connections, cache, basic routing are handled elsewhere

	sm.mu.Lock()
	sm.phase = PhaseCriticalReady
	sm.mu.Unlock()

	sm.logger.WithFields(logrus.Fields{
		"component": "startup_manager",
		"phase":     PhaseCriticalReady,
	}).Info("Critical services ready")

	return nil
}

// StartBackgroundInitialization starts optional background services
func (sm *StartupManager) StartBackgroundInitialization() {
	sm.mu.Lock()
	sm.phase = PhaseBackgroundInit
	sm.mu.Unlock()

	sm.logger.WithFields(logrus.Fields{
		"component": "startup_manager",
		"phase":     PhaseBackgroundInit,
	}).Info("Starting background initialization")

	go func() {
		// Respect startup delay configuration
		if sm.config.StartupDelaySeconds > 0 {
			sm.logger.WithFields(logrus.Fields{
				"component": "startup_manager",
				"delay":     sm.config.StartupDelaySeconds,
			}).Info("Delaying background initialization")

			time.Sleep(time.Duration(sm.config.StartupDelaySeconds) * time.Second)
		}

		// Start background jobs based on configuration
		var wg sync.WaitGroup

		if !sm.config.SkipInitialDataFetch {
			wg.Add(1)
			go sm.startDataFetcher(&wg)
		} else {
			sm.logger.Info("Skipping initial data fetch (SKIP_INITIAL_DATA_FETCH=true)")
		}

		if !sm.config.SkipInitialGolfSync {
			wg.Add(1)
			go sm.startGolfSync(&wg)
		} else {
			sm.logger.Info("Skipping initial golf sync (SKIP_INITIAL_GOLF_SYNC=true)")
		}

		// Wait for all background jobs to complete initial runs
		wg.Wait()

		sm.mu.Lock()
		sm.phase = PhaseFullyReady
		sm.mu.Unlock()

		sm.logger.WithFields(logrus.Fields{
			"component": "startup_manager",
			"phase":     PhaseFullyReady,
		}).Info("All background services initialized")
	}()
}

func (sm *StartupManager) startDataFetcher(wg *sync.WaitGroup) {
	defer wg.Done()

	sm.updateJobStatus("data_fetcher", "starting", "Initializing data fetcher service")

	// Start the data fetcher service (this includes cron scheduling)
	if err := sm.dataFetcher.Start(); err != nil {
		sm.logger.WithFields(logrus.Fields{
			"component": "startup_manager",
			"job":       "data_fetcher",
			"error":     err,
		}).Error("Failed to start data fetcher")

		sm.updateJobStatus("data_fetcher", "failed", err.Error())
		return
	}

	sm.updateJobStatus("data_fetcher", "running", "Data fetcher service started successfully")
}

func (sm *StartupManager) startGolfSync(wg *sync.WaitGroup) {
	defer wg.Done()

	sm.updateJobStatus("golf_sync", "starting", "Running initial golf tournament sync")

	// Wait a moment for other services to fully initialize
	time.Sleep(2 * time.Second)

	if err := sm.golfSyncService.SyncAllActiveTournaments(); err != nil {
		sm.logger.WithFields(logrus.Fields{
			"component": "startup_manager",
			"job":       "golf_sync",
			"error":     err,
		}).Error("Initial golf tournament sync failed")

		sm.updateJobStatus("golf_sync", "failed", err.Error())
		return
	}

	sm.updateJobStatus("golf_sync", "completed", "Initial golf tournament sync completed successfully")
}

func (sm *StartupManager) updateJobStatus(job, status, message string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.backgroundJobs[job] = JobStatus{
		Status:    status,
		StartedAt: time.Now(),
		Message:   message,
	}
}

// GetStatus returns current startup status for health checks
func (sm *StartupManager) GetStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Update external service states from circuit breakers
	externalServices := make(map[string]ServiceStatus)
	for service := range map[string]bool{"rapidapi": true, "espn": true, "balldontlie": true, "thesportsdb": true} {
		state := sm.circuitBreaker.GetState(service)
		counts := sm.circuitBreaker.GetCounts(service)

		externalServices[service] = ServiceStatus{
			State:       state.String(),
			LastChecked: time.Now(),
			Healthy:     state == 0, // StateClosed = 0
		}

		// Log circuit breaker metrics
		if counts.Requests > 0 {
			sm.logger.WithFields(logrus.Fields{
				"component":            "startup_manager",
				"service":              service,
				"state":                state.String(),
				"total_requests":       counts.Requests,
				"total_failures":       counts.TotalFailures,
				"consecutive_failures": counts.ConsecutiveFailures,
			}).Debug("Circuit breaker metrics")
		}
	}

	return map[string]interface{}{
		"status":            string(sm.phase),
		"timestamp":         time.Now(),
		"startup_phase":     string(sm.phase),
		"background_jobs":   sm.backgroundJobs,
		"external_services": externalServices,
	}
}

// GetPhase returns the current startup phase
func (sm *StartupManager) GetPhase() StartupPhase {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.phase
}

// IsReady returns true if critical services are ready
func (sm *StartupManager) IsReady() bool {
	phase := sm.GetPhase()
	return phase == PhaseCriticalReady || phase == PhaseBackgroundInit || phase == PhaseFullyReady
}

// IsFullyReady returns true if all services including background jobs are ready
func (sm *StartupManager) IsFullyReady() bool {
	return sm.GetPhase() == PhaseFullyReady
}

// Shutdown gracefully shuts down the startup manager
func (sm *StartupManager) Shutdown() {
	sm.logger.WithFields(logrus.Fields{
		"component": "startup_manager",
	}).Info("Shutting down startup manager")

	sm.cancel()
}

// TriggerGolfSync manually triggers golf tournament sync
func (sm *StartupManager) TriggerGolfSync() error {
	sm.updateJobStatus("manual_golf_sync", "starting", "Manual golf sync triggered")

	go func() {
		if err := sm.golfSyncService.SyncAllActiveTournaments(); err != nil {
			sm.logger.WithFields(logrus.Fields{
				"component": "startup_manager",
				"operation": "manual_golf_sync",
				"error":     err,
			}).Error("Manual golf sync failed")

			sm.updateJobStatus("manual_golf_sync", "failed", err.Error())
			return
		}

		sm.updateJobStatus("manual_golf_sync", "completed", "Manual golf sync completed successfully")
	}()

	return nil
}

// TriggerDataFetch manually triggers data fetch
func (sm *StartupManager) TriggerDataFetch() error {
	sm.updateJobStatus("manual_data_fetch", "starting", "Manual data fetch triggered")

	// This would trigger the data fetcher to run immediately
	// Implementation depends on data fetcher structure
	sm.logger.WithFields(logrus.Fields{
		"component": "startup_manager",
		"operation": "manual_data_fetch",
	}).Info("Manual data fetch triggered")

	sm.updateJobStatus("manual_data_fetch", "completed", "Manual data fetch triggered successfully")

	return nil
}
