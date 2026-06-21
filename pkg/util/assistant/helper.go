package assistant

import "time"

// TestHelper provides utilities for assistant test scenarios.
type TestHelper struct {
    CreatedAt time.Time
}

// NewTestHelper initializes a new TestHelper.
func NewTestHelper() *TestHelper {
    return &TestHelper{CreatedAt: time.Now()}
}

// Age returns how long the helper has existed.
func (h *TestHelper) Age() time.Duration {
    return time.Since(h.CreatedAt)
}
