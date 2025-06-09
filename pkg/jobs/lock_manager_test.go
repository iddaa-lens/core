package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockDB implements database.DBTX for testing
type MockDB struct {
	locks map[int64]bool
}

func NewMockDB() *MockDB {
	return &MockDB{
		locks: make(map[int64]bool),
	}
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	// Mock implementation for testing
	if len(args) > 0 {
		lockID := args[0].(int64)

		if query == "SELECT pg_try_advisory_lock($1)" {
			// Check if lock is already held
			if m.locks[lockID] {
				return &MockRow{value: false} // Lock already held
			}
			m.locks[lockID] = true
			return &MockRow{value: true} // Lock acquired
		}

		if query == "SELECT pg_advisory_unlock($1)" {
			// Release the lock
			wasHeld := m.locks[lockID]
			delete(m.locks, lockID)
			return &MockRow{value: wasHeld} // Return whether lock was held
		}
	}

	return &MockRow{value: false}
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// MockRow implements pgx.Row for testing
type MockRow struct {
	value interface{}
}

func (m *MockRow) Scan(dest ...interface{}) error {
	if len(dest) > 0 {
		switch v := dest[0].(type) {
		case *bool:
			*v = m.value.(bool)
		}
	}
	return nil
}

// TestLockManager tests the basic locking functionality
func TestLockManager(t *testing.T) {
	mockDB := NewMockDB()
	lockManager := NewPostgreSQLLockManager(mockDB)
	ctx := context.Background()

	// Test acquiring a lock
	acquired, err := lockManager.AcquireLock(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	if !acquired {
		t.Fatal("Expected to acquire lock but didn't")
	}

	// Test that the same lock cannot be acquired again
	acquired2, err := lockManager.AcquireLock(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to attempt second lock acquisition: %v", err)
	}
	if acquired2 {
		t.Fatal("Expected second lock acquisition to fail but it succeeded")
	}

	// Test checking if lock is held
	isLocked, err := lockManager.IsLocked(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to check lock status: %v", err)
	}
	if !isLocked {
		t.Fatal("Expected job to be locked but it wasn't")
	}

	// Test releasing the lock
	err = lockManager.ReleaseLock(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Test that lock can be acquired again after release
	acquired3, err := lockManager.AcquireLock(ctx, "test-job")
	if err != nil {
		t.Fatalf("Failed to acquire lock after release: %v", err)
	}
	if !acquired3 {
		t.Fatal("Expected to acquire lock after release but didn't")
	}
}

// TestLockGuard tests the RAII-style lock guard
func TestLockGuard(t *testing.T) {
	mockDB := NewMockDB()
	lockManager := NewPostgreSQLLockManager(mockDB)
	ctx := context.Background()

	// Test lock guard acquire and release
	guard := NewLockGuard(lockManager, "guard-test-job")

	acquired, err := guard.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire lock with guard: %v", err)
	}
	if !acquired {
		t.Fatal("Expected guard to acquire lock but didn't")
	}
	if !guard.IsAcquired() {
		t.Fatal("Guard should report as acquired")
	}

	// Test that another guard cannot acquire the same lock
	guard2 := NewLockGuard(lockManager, "guard-test-job")
	acquired2, err := guard2.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to attempt second guard acquisition: %v", err)
	}
	if acquired2 {
		t.Fatal("Expected second guard acquisition to fail but it succeeded")
	}

	// Test releasing with guard
	err = guard.Release(ctx)
	if err != nil {
		t.Fatalf("Failed to release lock with guard: %v", err)
	}
	if guard.IsAcquired() {
		t.Fatal("Guard should report as not acquired after release")
	}

	// Test that second guard can now acquire
	acquired3, err := guard2.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire lock with second guard after release: %v", err)
	}
	if !acquired3 {
		t.Fatal("Expected second guard to acquire lock after first release but didn't")
	}
}

// TestLockTimeout tests the timeout functionality
func TestLockTimeout(t *testing.T) {
	mockDB := NewMockDB()
	lockManager := NewPostgreSQLLockManager(mockDB)
	ctx := context.Background()

	// Acquire lock first
	acquired, err := lockManager.AcquireLock(ctx, "timeout-test-job")
	if err != nil {
		t.Fatalf("Failed to acquire initial lock: %v", err)
	}
	if !acquired {
		t.Fatal("Expected to acquire initial lock but didn't")
	}

	// Test timeout when lock is held
	start := time.Now()
	acquired2, err := lockManager.AcquireLockWithTimeout(ctx, "timeout-test-job", 200*time.Millisecond)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error but didn't get one")
	}
	if acquired2 {
		t.Fatal("Expected timeout to fail acquisition but it succeeded")
	}
	if duration < 150*time.Millisecond {
		t.Fatalf("Expected to wait for timeout but only waited %v", duration)
	}
}

// TestGenerateLockID tests that lock ID generation is consistent
func TestGenerateLockID(t *testing.T) {
	mockDB := NewMockDB()
	lockManager := NewPostgreSQLLockManager(mockDB).(*PostgreSQLLockManager)

	// Test that same job name generates same lock ID
	id1 := lockManager.generateLockID("test-job")
	id2 := lockManager.generateLockID("test-job")

	if id1 != id2 {
		t.Fatalf("Expected same lock ID for same job name, got %d and %d", id1, id2)
	}

	// Test that different job names generate different lock IDs
	id3 := lockManager.generateLockID("different-job")
	if id1 == id3 {
		t.Fatalf("Expected different lock IDs for different job names, both got %d", id1)
	}

	// Test that lock IDs are positive
	if id1 <= 0 {
		t.Fatalf("Expected positive lock ID, got %d", id1)
	}
}
