package periodic_scanner

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockScannerQueue implements the ScannerQueue interface for testing
type MockScannerQueue struct {
	mock.Mock
}

func (m *MockScannerQueue) AddAllToQueue() error {
	return m.Called().Error(0)
}

func TestMain(m *testing.M) {
	test_utils.IntegrationTestRun(m)
}

func resetPeriodicScanner() {

	if mainPeriodicScanner != nil {
		select {
		case <-mainPeriodicScanner.done:
			// Already closed
		default:
			close(mainPeriodicScanner.done)
		}
		if mainPeriodicScanner.ticker != nil {
			mainPeriodicScanner.ticker.Stop()
		}
		mainPeriodicScanner = nil
	}
}

func createTestSiteInfo(db *gorm.DB, interval int) error {
	siteInfo := models.SiteInfo{
		InitialSetup:         false,
		PeriodicScanInterval: interval,
		ConcurrentWorkers:    1,
	}
	return db.Create(&siteInfo).Error
}

func TestGetPeriodicScanInterval(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	t.Run("successful retrieval", func(t *testing.T) {
		assert.NoError(t, createTestSiteInfo(db, 300), "Failed to create test site info with 300 second interval")

		duration, err := getPeriodicScanInterval(db)
		assert.NoError(t, err, "Failed to retrieve periodic scan interval from database")
		assert.Equal(t, 300*time.Second, duration,
			"Periodic scan interval should be 300 seconds but got %v", duration)
	})

	t.Run("database error - no site info", func(t *testing.T) {
		db.Exec("DELETE FROM site_info")

		duration, err := getPeriodicScanInterval(db)
		assert.Error(t, err, "Expected error when no site info exists in database")
		assert.Equal(t, time.Duration(0), duration,
			"Duration should be zero when database error occurs, but got %v", duration)
	})
}

func TestInitializePeriodicScanner(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	t.Run("successful initialization with injection", func(t *testing.T) {
		defer resetPeriodicScanner()

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, createTestSiteInfo(db, 300), "Failed to create test site info with 300 second interval")
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue),
			"Failed to initialize periodic scanner with mock queue")

		// Verify initialization
		assert.NotNil(t, mainPeriodicScanner, "mainPeriodicScanner should not be nil after successful initialization")
		assert.NotNil(t, mainPeriodicScanner.scannerQueue, "Scanner queue should not be nil after initialization")
		assert.Equal(t, mockQueue, mainPeriodicScanner.scannerQueue, "Scanner should use the injected mock queue instance")

		// Verify ticker is set up
		mainPeriodicScanner.tickerLocker.Lock()
		tickerExists := mainPeriodicScanner.ticker != nil
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.True(t, tickerExists, "Ticker should be created and set up after scanner initialization")
	})

	t.Run("backward compatibility with original function", func(t *testing.T) {
		defer resetPeriodicScanner()

		assert.NoError(t, createTestSiteInfo(db, 300), "Failed to create test site info with 300 second interval")
		assert.NoError(t, InitializePeriodicScanner(db),
			"Failed to initialize periodic scanner using original function")

		// Verify it uses RealScannerQueue
		assert.NotNil(t, mainPeriodicScanner,
			"mainPeriodicScanner should be initialized by original InitializePeriodicScanner function")
		assert.IsType(t, &RealScannerQueue{}, mainPeriodicScanner.scannerQueue,
			"Original InitializePeriodicScanner should use RealScannerQueue by default")
	})

	t.Run("double initialization error", func(t *testing.T) {
		defer resetPeriodicScanner()

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, createTestSiteInfo(db, 300), "Failed to create test site info with 300 second interval")
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue),
			"Failed first initialization for double initialization test")

		err := InitializePeriodicScannerWithQueue(db, mockQueue)
		assert.Error(t, err, "Second initialization attempt should return an error")
		assert.Contains(t, err.Error(), "already been initialized",
			"Double initialization error should contain 'already been initialized' message")
	})
}

func TestScanIntervalRunnerWithMocking(t *testing.T) {
	t.Run("runner calls queue on ticker events", func(t *testing.T) {
		mockQueue := &MockScannerQueue{}

		// Use a channel to synchronize and count calls
		callChan := make(chan struct{}, 5) // Buffer for multiple calls
		mockQueue.On("AddAllToQueue").Return(nil).Run(func(args mock.Arguments) {
			select {
			case callChan <- struct{}{}:
			default:
				// Channel full, but that's okay
			}
		}).Maybe()

		ps := &periodicScanner{
			ticker:         time.NewTicker(50 * time.Millisecond),
			ticker_changed: make(chan bool, 1),
			done:           make(chan struct{}),
			tickerLocker:   sync.Mutex{},
			scannerQueue:   mockQueue,
		}

		// Start runner
		go ps.scanIntervalRunner()

		// Wait for at least one call with timeout
		select {
		case <-callChan:
			// Success - at least one call received
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Expected at least one call to AddAllToQueue within 200ms timeout, but ticker events were not processed")
		}

		// Proper cleanup - stop ticker first, then close done
		ps.ticker.Stop()
		close(ps.done)

		// Give time for goroutine to finish
		time.Sleep(50 * time.Millisecond)

		// Verify the queue was called as expected
		mockQueue.AssertExpectations(t)
	})

	t.Run("runner handles queue errors gracefully", func(t *testing.T) {
		mockQueue := &MockScannerQueue{}
		// Mock queue to return an error
		mockQueue.On("AddAllToQueue").Return(errors.New("queue error")).Maybe()

		ps := &periodicScanner{
			ticker:         time.NewTicker(30 * time.Millisecond),
			ticker_changed: make(chan bool, 1),
			done:           make(chan struct{}),
			tickerLocker:   sync.Mutex{},
			scannerQueue:   mockQueue,
		}

		// Start runner and let it run briefly
		go ps.scanIntervalRunner()
		time.Sleep(80 * time.Millisecond)

		// Proper cleanup
		ps.ticker.Stop()
		close(ps.done)
		time.Sleep(50 * time.Millisecond)

		// Test passes if no panic occurred - errors should be logged gracefully
		mockQueue.AssertExpectations(t)
	})

	t.Run("runner responds to shutdown signal", func(t *testing.T) {
		mockQueue := &MockScannerQueue{}

		ps := &periodicScanner{
			ticker:         nil, // No ticker to avoid timing issues
			ticker_changed: make(chan bool, 1),
			done:           make(chan struct{}),
			tickerLocker:   sync.Mutex{},
			scannerQueue:   mockQueue,
		}

		runnerDone := make(chan bool)
		go func() {
			ps.scanIntervalRunner()
			close(runnerDone)
		}()

		close(ps.done)

		select {
		case <-runnerDone:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("scanIntervalRunner goroutine did not exit within 1 second after closing done channel")
		}

		mockQueue.AssertExpectations(t)
	})

	t.Run("runner responds to ticker changes", func(t *testing.T) {
		mockQueue := &MockScannerQueue{}

		ps := &periodicScanner{
			ticker:         nil, // Start without ticker
			ticker_changed: make(chan bool, 1),
			done:           make(chan struct{}),
			tickerLocker:   sync.Mutex{},
			scannerQueue:   mockQueue,
		}

		runnerDone := make(chan bool)
		go func() {
			ps.scanIntervalRunner()
			close(runnerDone)
		}()

		// Send ticker change signal
		select {
		case ps.ticker_changed <- true:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Could not send ticker change signal within 100ms - channel may be blocked")
		}

		// Clean shutdown
		close(ps.done)

		// Wait for completion
		select {
		case <-runnerDone:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Fatal("scanIntervalRunner did not exit within 500ms after receiving ticker change signal and shutdown")
		}

		// Test passes if no deadlock occurs
		mockQueue.AssertExpectations(t)
	})
}

func TestRealScannerQueue(t *testing.T) {
	t.Run("real scanner queue interface compliance", func(t *testing.T) {
		queue := &RealScannerQueue{}

		// Just verify it implements the interface correctly
		var _ ScannerQueue = queue

		// Test that it doesn't panic when created
		assert.NotNil(t, queue, "RealScannerQueue instance should not be nil after creation")

		// We don't test the actual call since it requires external setup
	})
}

func TestChangePeriodicScanInterval(t *testing.T) {
	t.Run("changes interval from disabled to enabled", func(t *testing.T) {
		defer resetPeriodicScanner()

		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 0), "Failed to create site info with disabled scanner")

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize scanner")

		// Verify ticker is nil initially (disabled)
		mainPeriodicScanner.tickerLocker.Lock()
		initialTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.Nil(t, initialTicker, "Ticker should be nil when scanner is disabled")

		// Change interval to enabled
		ChangePeriodicScanInterval(100 * time.Millisecond)

		// Verify new ticker is created
		mainPeriodicScanner.tickerLocker.Lock()
		newTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.NotNil(t, newTicker, "Ticker should be created after enabling scanner")
	})

	t.Run("changes interval from enabled to different value", func(t *testing.T) {
		defer resetPeriodicScanner()

		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 500), "Failed to create site info with 500ms interval")

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize scanner")

		mainPeriodicScanner.tickerLocker.Lock()
		oldTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.NotNil(t, oldTicker, "Ticker should exist initially")

		// Change interval
		ChangePeriodicScanInterval(200 * time.Millisecond)

		// Verify ticker was replaced
		mainPeriodicScanner.tickerLocker.Lock()
		newTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.NotNil(t, newTicker, "New ticker should exist")
		assert.NotSame(t, oldTicker, newTicker, "Ticker should be replaced with new instance")
	})

	t.Run("changes interval from enabled to disabled", func(t *testing.T) {
		defer resetPeriodicScanner()

		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 300), "Failed to create site info with 300ms interval")

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize scanner")

		// Verify ticker exists
		mainPeriodicScanner.tickerLocker.Lock()
		oldTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.NotNil(t, oldTicker, "Ticker should exist initially")

		// Disable scanner
		ChangePeriodicScanInterval(0)

		// Verify ticker is nil
		mainPeriodicScanner.tickerLocker.Lock()
		newTicker := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.Nil(t, newTicker, "Ticker should be nil after disabling scanner")
	})

	t.Run("handles nil scanner gracefully", func(t *testing.T) {
		// Ensure scanner is nil
		resetPeriodicScanner()

		// Should not panic when scanner is nil
		assert.NotPanics(t, func() {
			ChangePeriodicScanInterval(100 * time.Millisecond)
		}, "ChangePeriodicScanInterval should not panic when scanner is nil")
	})
}

func TestShutdownPeriodicScanner(t *testing.T) {
	t.Run("graceful shutdown stops ticker and goroutine", func(t *testing.T) {
		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 50), "Failed to create site info with 50ms interval")

		mockQueue := &MockScannerQueue{}
		mockQueue.On("AddAllToQueue").Return(nil).Maybe()

		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize scanner")

		// Wait a bit for scanner to start
		time.Sleep(20 * time.Millisecond)

		// Verify scanner is running
		assert.NotNil(t, mainPeriodicScanner, "Scanner should be initialized")
		mainPeriodicScanner.tickerLocker.Lock()
		tickerBefore := mainPeriodicScanner.ticker
		mainPeriodicScanner.tickerLocker.Unlock()
		assert.NotNil(t, tickerBefore, "Ticker should exist before shutdown")

		// Shutdown
		assert.NotPanics(t, func() {
			ShutdownPeriodicScanner()
		}, "ShutdownPeriodicScanner should not panic")

		// Verify scanner was reset
		assert.Nil(t, mainPeriodicScanner, "mainPeriodicScanner should be nil after shutdown")

		mockQueue.AssertExpectations(t)
	})

	t.Run("multiple shutdowns are safe", func(t *testing.T) {
		defer resetPeriodicScanner()

		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 100), "Failed to create site info")

		mockQueue := &MockScannerQueue{}
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize scanner")

		// First shutdown
		assert.NotPanics(t, func() {
			ShutdownPeriodicScanner()
		}, "First shutdown should not panic")

		// Second shutdown should also be safe
		assert.NotPanics(t, func() {
			ShutdownPeriodicScanner()
		}, "Second shutdown should not panic")

		assert.Nil(t, mainPeriodicScanner, "Scanner should remain nil after multiple shutdowns")

		mockQueue.AssertExpectations(t)
	})

	t.Run("shutdown when scanner is nil is safe", func(t *testing.T) {
		resetPeriodicScanner()

		assert.NotPanics(t, func() {
			ShutdownPeriodicScanner()
		}, "Shutdown should not panic when scanner is already nil")
	})
}

func TestPeriodicScannerIntegration(t *testing.T) {
	t.Run("full lifecycle: init, change interval, shutdown", func(t *testing.T) {
		defer resetPeriodicScanner()

		db := test_utils.DatabaseTest(t)
		assert.NoError(t, createTestSiteInfo(db, 200), "Failed to create site info")

		mockQueue := &MockScannerQueue{}
		mockQueue.On("AddAllToQueue").Return(nil).Maybe()

		// Initialize
		assert.NoError(t, InitializePeriodicScannerWithQueue(db, mockQueue), "Failed to initialize")
		assert.NotNil(t, mainPeriodicScanner, "Scanner should be initialized")

		// Change interval
		ChangePeriodicScanInterval(100 * time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		// Disable
		ChangePeriodicScanInterval(0)
		time.Sleep(30 * time.Millisecond)

		// Re-enable
		ChangePeriodicScanInterval(50 * time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		// Shutdown
		assert.NotPanics(t, func() {
			ShutdownPeriodicScanner()
		}, "Shutdown should not panic")

		assert.Nil(t, mainPeriodicScanner, "Scanner should be nil after shutdown")

		mockQueue.AssertExpectations(t)
	})
}
