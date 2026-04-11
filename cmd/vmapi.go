// Package cmd provides CLI commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/vm"
	"github.com/spf13/cobra"
)

// vmapiCmd represents the VM API server command
var vmapiCmd = &cobra.Command{
	Use:   "vmapi",
	Short: "Start REST API server for VM management",
	Long: `Start an HTTP server that provides a REST API for VM management.
This is designed to be called by trooper.stsgym.com or other orchestrators.

Endpoints:
  GET    /api/v1/vms           - List all VMs
  GET    /api/v1/vms/:name     - Get VM status
  POST   /api/v1/vms           - Create VM
  POST   /api/v1/vms/:name/start   - Start VM
  POST   /api/v1/vms/:name/stop    - Stop VM
  POST   /api/v1/vms/:name/restart  - Restart VM
  DELETE /api/v1/vms/:name     - Delete VM
  GET    /api/v1/vms/:name/snapshots - List snapshots
  POST   /api/v1/vms/:name/snapshots - Create snapshot
  POST   /api/v1/vms/:name/snapshots/:snap/revert - Revert to snapshot
  DELETE /api/v1/vms/:name/snapshots/:snap - Delete snapshot
  GET    /health               - Health check`,
	RunE: runVMAPI,
}

func init() {
	rootCmd.AddCommand(vmapiCmd)
	vmapiCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	vmapiCmd.Flags().StringP("host", "H", "127.0.0.1", "Host to bind to")
	vmapiCmd.Flags().StringP("token", "t", "", "API token for authentication")
	vmapiCmd.Flags().Bool("tls", false, "Enable TLS")
	vmapiCmd.Flags().String("cert", "", "TLS certificate path")
	vmapiCmd.Flags().String("key", "", "TLS key path")
}

var (
	vmManager vm.Manager
	apiToken  string
)

// API Response types

// VMListResponse represents a list of VMs
type VMListResponse struct {
	VMs []VMInfoAPI `json:"vms"`
}

// VMInfoAPI represents VM information for API responses
type VMInfoAPI struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IP        string `json:"ip,omitempty"`
	MemoryMB  int    `json:"memory_mb"`
	VCPUs     int    `json:"vcpus"`
	DiskGB    int    `json:"disk_gb,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// VMDetailResponse represents detailed VM status
type VMDetailResponse struct {
	Name        string          `json:"name"`
	State       string          `json:"state"`
	IP          string          `json:"ip,omitempty"`
	MemoryMB    int             `json:"memory_mb"`
	VCPUs       int             `json:"vcpus"`
	DiskGB      int             `json:"disk_gb,omitempty"`
	CPUUsage    float64         `json:"cpu_usage,omitempty"`
	MemoryUsage float64         `json:"memory_usage,omitempty"`
	Uptime      string          `json:"uptime,omitempty"`
	Disks       []DiskInfoAPI   `json:"disks,omitempty"`
	Interfaces  []InterfaceInfo `json:"interfaces,omitempty"`
}

// DiskInfoAPI represents disk information
type DiskInfoAPI struct {
	Device string `json:"device"`
	SizeGB int    `json:"size_gb"`
	Type   string `json:"type"`
}

// InterfaceInfo represents network interface information
type InterfaceInfo struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IP   string `json:"ip,omitempty"`
	Type string `json:"type"`
}

// CreateVMRequest represents a VM creation request
type CreateVMRequest struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	MemoryMB int    `json:"memory_mb,omitempty"`
	VCPUs    int    `json:"vcpus,omitempty"`
	DiskGB   int    `json:"disk_gb,omitempty"`
	Network  string `json:"network,omitempty"`
	UserData string `json:"user_data,omitempty"` // cloud-init user-data
}

// CreateVMResponse represents a VM creation response
type CreateVMResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// SnapshotAPIResponse represents snapshot information
type SnapshotAPIResponse struct {
	Name      string `json:"name"`
	VM        string `json:"vm"`
	CreatedAt string `json:"created_at,omitempty"`
	Current   bool   `json:"current"`
}

// SnapshotListAPIResponse represents a list of snapshots
type SnapshotListAPIResponse struct {
	VM        string                 `json:"vm"`
	Snapshots []SnapshotAPIResponse `json:"snapshots"`
}

// ActionResponse represents a generic action response
type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func runVMAPI(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	apiToken, _ = cmd.Flags().GetString("token")

	// Create VM manager
	var err error
	vmManager, err = vm.NewLibvirtManager(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %w", err)
	}

	// Setup routes with authentication middleware
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", handleAPIHealth)

	// VM endpoints (auth required)
	mux.HandleFunc("/api/v1/vms", withAuth(handleVMs))
	mux.HandleFunc("/api/v1/vms/", withAuth(handleVMByName))

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("VM API server starting on %s", addr)
	authStatus := "none"
	if apiToken != "" {
		authStatus = "token required"
	}
	log.Printf("Authentication: %s", authStatus)

	// Check TLS
	tls, _ := cmd.Flags().GetBool("tls")
	if tls {
		cert, _ := cmd.Flags().GetString("cert")
		key, _ := cmd.Flags().GetString("key")
		if cert == "" || key == "" {
			return fmt.Errorf("TLS requires --cert and --key")
		}
		return http.ListenAndServeTLS(addr, cert, key, mux)
	}

	return http.ListenAndServe(addr, mux)
}

// withAuth wraps a handler with token authentication
func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no token configured
		if apiToken == "" {
			next(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			sendAPIError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Extract token
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			sendAPIError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		if parts[1] != apiToken {
			sendAPIError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		next(w, r)
	}
}

func handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"version":   "2.0.0",
		"type":      "cicerone-vmapi",
		"libvirt":   "connected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVMs handles /api/v1/vms (list and create)
func handleVMs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listVMs(w, r)
	case http.MethodPost:
		createVM(w, r)
	default:
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleVMByName handles /api/v1/vms/:name/* endpoints
func handleVMByName(w http.ResponseWriter, r *http.Request) {
	// Extract VM name and action from path
	// Path format: /api/v1/vms/:name or /api/v1/vms/:name/action
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/vms/")

	// Split path into parts
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		sendAPIError(w, http.StatusBadRequest, "VM name required")
		return
	}

	vmName := parts[0]

	// Route based on remaining path
	switch {
	case len(parts) == 1:
		// /api/v1/vms/:name
		handleSingleVM(w, r, vmName)
	case parts[1] == "start":
		startVM(w, r, vmName)
	case parts[1] == "stop":
		stopVM(w, r, vmName)
	case parts[1] == "restart":
		restartVM(w, r, vmName)
	case parts[1] == "snapshots":
		handleSnapshots(w, r, vmName, parts[2:])
	default:
		sendAPIError(w, http.StatusNotFound, "endpoint not found")
	}
}

// listVMs returns a list of all VMs
func listVMs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vms, err := vmManager.List(ctx)
	if err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list VMs: %v", err))
		return
	}

	result := make([]VMInfoAPI, len(vms))
	for i, v := range vms {
		result[i] = VMInfoAPI{
			Name:     v.Name,
			State:    string(v.State),
			IP:       v.IP,
			MemoryMB: v.Memory,
			VCPUs:    v.VCPUs,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(VMListResponse{VMs: result})
}

// handleSingleVM handles GET and DELETE for a single VM
func handleSingleVM(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		getVMStatus(w, r, name)
	case http.MethodDelete:
		deleteVM(w, r, name)
	default:
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getVMStatus returns detailed status for a VM
func getVMStatus(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	status, err := vmManager.Status(ctx, name)
	if err != nil {
		sendAPIError(w, http.StatusNotFound, fmt.Sprintf("VM not found: %v", err))
		return
	}

	response := VMDetailResponse{
		Name:     status.Name,
		State:    string(status.State),
		IP:       status.IP,
		MemoryMB: status.Memory,
		VCPUs:    status.VCPUs,
		Uptime:   status.Uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// createVM creates a new VM
func createVM(w http.ResponseWriter, r *http.Request) {
	var req CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Name == "" {
		sendAPIError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Image == "" {
		sendAPIError(w, http.StatusBadRequest, "image is required")
		return
	}

	// Set defaults
	if req.MemoryMB == 0 {
		req.MemoryMB = 8192
	}
	if req.VCPUs == 0 {
		req.VCPUs = 4
	}
	if req.DiskGB == 0 {
		req.DiskGB = 50
	}
	if req.Network == "" {
		req.Network = "default"
	}

	// Create VM config
	config := &vm.VMConfig{
		Name:     req.Name,
		Image:    req.Image,
		Memory:   req.MemoryMB,
		VCPUs:    req.VCPUs,
		DiskSize: req.DiskGB,
		Network:  req.Network,
	}

	// Create VM
	ctx := r.Context()
	if _, err := vmManager.Create(ctx, config); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create VM: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateVMResponse{
		Name:    req.Name,
		Status:  "creating",
		Message: "VM creation initiated",
	})
}

// startVM starts a VM
func startVM(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	if err := vmManager.Start(ctx, name); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start VM: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("VM %s started", name),
	})
}

// stopVM stops a VM
func stopVM(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	ctx := r.Context()
	if err := vmManager.Stop(ctx, name, force); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stop VM: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("VM %s stopped", name),
	})
}

// restartVM restarts a VM
func restartVM(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	if err := vmManager.Restart(ctx, name); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to restart VM: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("VM %s restarted", name),
	})
}

// deleteVM deletes a VM
func deleteVM(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodDelete {
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	if err := vmManager.Delete(ctx, name); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete VM: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("VM %s deleted", name),
	})
}

// handleSnapshots handles snapshot operations
func handleSnapshots(w http.ResponseWriter, r *http.Request, vmName string, parts []string) {
	switch r.Method {
	case http.MethodGet:
		listSnapshots(w, r, vmName)
	case http.MethodPost:
		if len(parts) > 1 && parts[1] == "revert" {
			revertSnapshot(w, r, vmName, parts[0])
		} else {
			createSnapshot(w, r, vmName)
		}
	case http.MethodDelete:
		if len(parts) == 0 {
			sendAPIError(w, http.StatusBadRequest, "snapshot name required")
			return
		}
		deleteSnapshot(w, r, vmName, parts[0])
	default:
		sendAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listSnapshots lists snapshots for a VM
func listSnapshots(w http.ResponseWriter, r *http.Request, vmName string) {
	ctx := r.Context()
	snapshots, err := vmManager.SnapshotList(ctx, vmName)
	if err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list snapshots: %v", err))
		return
	}

	result := make([]SnapshotAPIResponse, len(snapshots))
	for i, s := range snapshots {
		result[i] = SnapshotAPIResponse{
			Name:      s.Name,
			VM:        vmName,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			Current:   s.Current,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SnapshotListAPIResponse{
		VM:        vmName,
		Snapshots: result,
	})
}

// createSnapshot creates a new snapshot
func createSnapshot(w http.ResponseWriter, r *http.Request, vmName string) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	ctx := r.Context()
	if err := vmManager.Snapshot(ctx, vmName, name, ""); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create snapshot: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(SnapshotAPIResponse{
		Name:      name,
		VM:        vmName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// revertSnapshot reverts to a snapshot
func revertSnapshot(w http.ResponseWriter, r *http.Request, vmName, snapName string) {
	ctx := r.Context()
	if err := vmManager.SnapshotRevert(ctx, vmName, snapName); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to revert snapshot: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("Reverted VM %s to snapshot %s", vmName, snapName),
	})
}

// deleteSnapshot deletes a snapshot
func deleteSnapshot(w http.ResponseWriter, r *http.Request, vmName, snapName string) {
	ctx := r.Context()
	if err := vmManager.SnapshotDelete(ctx, vmName, snapName); err != nil {
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete snapshot: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("Deleted snapshot %s for VM %s", snapName, vmName),
	})
}

// sendAPIError sends an error response
func sendAPIError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: message,
	})
}

// parseIntParam parses an integer query parameter
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}