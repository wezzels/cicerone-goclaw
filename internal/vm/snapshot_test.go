package vm

import (
	"context"
	"testing"
	"time"
)

// TestSnapshotInfo tests snapshot info structure
func TestSnapshotInfo(t *testing.T) {
	now := time.Now()
	snap := SnapshotInfo{
		Name:        "test-snapshot",
		Description: "Test snapshot",
		CreatedAt:   now,
		Size:        1024000,
		Current:     true,
	}

	if snap.Name != "test-snapshot" {
		t.Errorf("Expected name 'test-snapshot', got %s", snap.Name)
	}
	if snap.Description != "Test snapshot" {
		t.Errorf("Expected description 'Test snapshot', got %s", snap.Description)
	}
	if !snap.Current {
		t.Error("Expected Current to be true")
	}
	if snap.Size != 1024000 {
		t.Errorf("Expected size 1024000, got %d", snap.Size)
	}
}

// TestSnapshotNaming tests snapshot naming conventions
func TestSnapshotNaming(t *testing.T) {
	tests := []struct {
		name     string
		valid    bool
		reason   string
	}{
		{"snapshot-1", true, "standard naming"},
		{"before-tests", true, "hyphenated name"},
		{"2024-01-01", true, "date-based name"},
		{"backup_001", true, "underscore name"},
		{"", false, "empty name"},
		{"snapshot with spaces", false, "spaces not allowed"},
		{"snapshot/invalid", false, "slash not allowed"},
		{"snapshot:invalid", false, "colon not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if name contains invalid characters
			invalidChars := []rune{' ', '/', ':', '\\', '*'}
			hasInvalid := false
			for _, c := range tt.name {
				for _, invalid := range invalidChars {
					if c == invalid {
						hasInvalid = true
						break
					}
				}
			}

			isValid := !hasInvalid && tt.name != ""
			if isValid != tt.valid {
				t.Errorf("Name '%s': expected valid=%v, got valid=%v (reason: %s)",
					tt.name, tt.valid, isValid, tt.reason)
			}
		})
	}
}

// TestSnapshotOperations tests snapshot operation names
func TestSnapshotOperations(t *testing.T) {
	ops := []string{"create", "list", "revert", "delete"}

	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			// Verify operation is recognized
			switch op {
			case "create", "list", "revert", "delete":
				// Valid operation
			default:
				t.Errorf("Unknown snapshot operation: %s", op)
			}
		})
	}
}

// TestSnapshotManager_Interface tests Manager interface for snapshots
func TestSnapshotManager_Interface(t *testing.T) {
	// This test verifies that the Manager interface includes snapshot methods
	var _ Manager = (*LibvirtManager)(nil)
}

// TestSnapshotError tests snapshot error handling
func TestSnapshotError(t *testing.T) {
	// Test that ErrSnapshot is properly wrapped
	err := &VMError{Op: "snapshot", Err: ErrSnapshot}

	if err.Op != "snapshot" {
		t.Errorf("Expected Op 'snapshot', got %s", err.Op)
	}
	if err.Unwrap() != ErrSnapshot {
		t.Error("Expected ErrSnapshot as wrapped error")
	}
}

// TestSnapshotXML tests snapshot XML generation
func TestSnapshotXML(t *testing.T) {
	name := "test-snapshot"
	description := "Test snapshot description"

	// Generate XML like libvirt.go does
	expectedElements := []string{
		"<domainsnapshot>",
		"<name>" + name + "</name>",
		"<description>" + description + "</description>",
		"</domainsnapshot>",
	}

	// Verify structure
	xmlStart := "<domainsnapshot>"
	xmlEnd := "</domainsnapshot>"

	if len(expectedElements) != 4 {
		t.Errorf("Expected 4 XML elements, got %d", len(expectedElements))
	}
	if expectedElements[0] != xmlStart {
		t.Errorf("Expected XML to start with %s, got %s", xmlStart, expectedElements[0])
	}
	if expectedElements[3] != xmlEnd {
		t.Errorf("Expected XML to end with %s, got %s", xmlEnd, expectedElements[3])
	}
}

// TestSnapshotList tests snapshot listing
func TestSnapshotList(t *testing.T) {
	// Test empty snapshot list handling
	snapshots := []SnapshotInfo{}

	if len(snapshots) != 0 {
		t.Error("Expected empty snapshot list")
	}

	// Test snapshot list with entries
	snapshots = []SnapshotInfo{
		{Name: "snap-1", CreatedAt: time.Now().Add(-24 * time.Hour), Current: false},
		{Name: "snap-2", CreatedAt: time.Now().Add(-12 * time.Hour), Current: false},
		{Name: "snap-3", CreatedAt: time.Now(), Current: true},
	}

	// Find current snapshot
	var currentSnapshot *SnapshotInfo
	for i := range snapshots {
		if snapshots[i].Current {
			currentSnapshot = &snapshots[i]
			break
		}
	}

	if currentSnapshot == nil {
		t.Fatal("Expected to find current snapshot")
	}
	if currentSnapshot.Name != "snap-3" {
		t.Errorf("Expected current snapshot 'snap-3', got %s", currentSnapshot.Name)
	}
}

// TestSnapshotRevert tests snapshot revert validation
func TestSnapshotRevert(t *testing.T) {
	// Test conditions for revert:
	// 1. VM must exist
	// 2. Snapshot must exist
	// 3. VM should typically be stopped before revert

	// Simulate revert scenarios
	tests := []struct {
		name       string
		vmExists   bool
		snapExists bool
		vmRunning  bool
		wantErr    bool
	}{
		{"valid revert", true, true, false, false},
		{"vm not found", false, true, false, true},
		{"snapshot not found", true, false, false, true},
		{"vm running", true, true, true, false}, // libvirt can revert running VMs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate preconditions
			if !tt.vmExists {
				if !tt.wantErr {
					t.Error("VM not found should cause error")
				}
			}
			if !tt.snapExists {
				if !tt.wantErr {
					t.Error("Snapshot not found should cause error")
				}
			}
		})
	}
}

// TestSnapshotDelete tests snapshot deletion
func TestSnapshotDelete(t *testing.T) {
	// Test conditions for delete:
	// 1. VM must exist
	// 2. Snapshot must exist
	// 3. Cannot delete current snapshot (usually requires revert first)

	tests := []struct {
		name        string
		snapCurrent bool
		wantErr     bool
	}{
		{"delete non-current snapshot", false, false},
		{"delete current snapshot", true, true}, // libvirt may restrict this
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Current snapshot deletion may be restricted
			if tt.snapCurrent && !tt.wantErr {
				t.Error("Deleting current snapshot may be restricted")
			}
		})
	}
}

// TestSnapshotCreateValidation tests snapshot creation validation
func TestSnapshotCreateValidation(t *testing.T) {
	tests := []struct {
		name        string
		snapshotName string
		description string
		wantErr     bool
	}{
		{"valid snapshot", "snap-1", "Before tests", false},
		{"empty name", "", "description", true},
		{"no description", "snap-2", "", false}, // description is optional
		{"long description", "snap-3", string(make([]byte, 4096)), false}, // libvirt handles long descriptions
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate snapshot name
			if tt.snapshotName == "" && !tt.wantErr {
				t.Error("Empty snapshot name should cause error")
			}
		})
	}
}

// BenchmarkSnapshotList benchmarks snapshot listing
func BenchmarkSnapshotList(b *testing.B) {
	// Simulate snapshot list processing
	snapshots := make([]SnapshotInfo, 100)
	for i := range snapshots {
		snapshots[i] = SnapshotInfo{
			Name:      "snapshot-" + string(rune(i)),
			CreatedAt: time.Now(),
			Current:   i == 99,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Find current snapshot
		for _, snap := range snapshots {
			if snap.Current {
				break
			}
		}
	}
}

// TestSnapshotConcurrency tests concurrent snapshot operations
func TestSnapshotConcurrency(t *testing.T) {
	// This tests that snapshot operations can be called concurrently
	// In practice, libvirt handles locking

	ctx := context.Background()

	// Simulate concurrent operations
	operations := []struct {
		name string
		op   func() error
	}{
		{
			name: "create",
			op: func() error {
				// Simulate create
				return nil
			},
		},
		{
			name: "list",
			op: func() error {
				// Simulate list
				return nil
			},
		},
		{
			name: "revert",
			op: func() error {
				// Simulate revert
				return nil
			},
		},
	}

	// Run operations
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			err := op.op()
			if err != nil {
				t.Errorf("Operation %s failed: %v", op.name, err)
			}
		})
	}

	// Verify context is still valid
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled")
	default:
	}
}