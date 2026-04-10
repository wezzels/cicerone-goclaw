//go:build libvirt

// Package vm provides VM management for workspace deployment.
package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sshpkg "github.com/crab-meat-repos/cicerone-goclaw/internal/ssh"
	"libvirt.org/go/libvirt"
)

// LibvirtManager implements Manager using libvirt/QEMU.
type LibvirtManager struct {
	conn       *libvirt.Connect
	opts       *ManagerOptions
	sshKeyPath string
	keyManager *KeyManager
}

// NewLibvirtManager creates a new libvirt-based VM manager.
func NewLibvirtManager(opts *ManagerOptions) (*LibvirtManager, error) {
	if opts == nil {
		opts = DefaultManagerOptions()
	}

	// Build connection URI
	uri := opts.URI
	if uri == "" {
		uri = "qemu:///system"
	}

	// Connect to libvirt
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, &VMError{Op: "connect", Err: fmt.Errorf("failed to connect to libvirt: %w", err)}
	}

	// Create key manager
	keyMgr, err := NewKeyManager()
	if err != nil {
		// Non-fatal, key manager is optional
		keyMgr = nil
	}

	return &LibvirtManager{
		conn:       conn,
		opts:       opts,
		keyManager: keyMgr,
	}, nil
}

// Close closes the libvirt connection.
func (m *LibvirtManager) Close() error {
	if m.conn != nil {
		_, err := m.conn.Close()
		return err
	}
	return nil
}

// Create creates a new VM from the configuration.
func (m *LibvirtManager) Create(ctx context.Context, cfg *VMConfig) (*VMInfo, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Check if VM already exists
	exists, err := m.Exists(ctx, cfg.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	if m.opts.DryRun {
		return &VMInfo{Name: cfg.Name, State: StateCreating}, nil
	}

	// Create VM XML definition
	xml, err := m.generateDomainXML(cfg)
	if err != nil {
		return nil, &VMError{Op: "create", Err: err}
	}

	// Define and start VM
	dom, err := m.conn.DomainDefineXML(xml)
	if err != nil {
		return nil, &VMError{Op: "create", Err: fmt.Errorf("failed to define domain: %w", err)}
	}
	defer dom.Free()

	// Start the VM
	if cfg.AutoStart {
		if err := dom.Create(); err != nil {
			return nil, &VMError{Op: "create", Err: fmt.Errorf("failed to start domain: %w", err)}
		}
	}

	return m.Status(ctx, cfg.Name)
}

// Delete removes a VM.
func (m *LibvirtManager) Delete(ctx context.Context, name string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	// Check if running
	state, _, err := dom.GetState()
	if err != nil {
		return &VMError{Op: "delete", Err: err}
	}

	// Stop if running
	if state == libvirt.DOMAIN_RUNNING {
		if err := dom.Destroy(); err != nil {
			return &VMError{Op: "delete", Err: fmt.Errorf("failed to destroy domain: %w", err)}
		}
	}

	// Undefine
	if err := dom.UndefineFlags(libvirt.DOMAIN_UNDEFINE_NVRAM); err != nil {
		return &VMError{Op: "delete", Err: fmt.Errorf("failed to undefine domain: %w", err)}
	}

	return nil
}

// Start starts a VM.
func (m *LibvirtManager) Start(ctx context.Context, name string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return &VMError{Op: "start", Err: err}
	}

	if state == libvirt.DOMAIN_RUNNING {
		return nil // Already running
	}

	if err := dom.Create(); err != nil {
		return &VMError{Op: "start", Err: err}
	}

	return nil
}

// Stop stops a VM.
func (m *LibvirtManager) Stop(ctx context.Context, name string, force bool) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return &VMError{Op: "stop", Err: err}
	}

	if state != libvirt.DOMAIN_RUNNING {
		return ErrNotRunning
	}

	if force {
		return dom.Destroy()
	}

	// Graceful shutdown
	return dom.Shutdown()
}

// Restart restarts a VM.
func (m *LibvirtManager) Restart(ctx context.Context, name string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	if err := dom.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT); err != nil {
		// Fallback: stop then start
		if err := m.Stop(ctx, name, false); err != nil {
			return err
		}
		// Wait for shutdown
		time.Sleep(2 * time.Second)
		return m.Start(ctx, name)
	}

	return nil
}

// Status returns current VM status.
func (m *LibvirtManager) Status(ctx context.Context, name string) (*VMInfo, error) {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	return m.domainToInfo(dom)
}

// List returns all VMs.
func (m *LibvirtManager) List(ctx context.Context) ([]*VMInfo, error) {
	doms, err := m.conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, &VMError{Op: "list", Err: err}
	}

	var infos []*VMInfo
	for _, dom := range doms {
		info, err := m.domainToInfo(&dom)
		if err != nil {
			dom.Free()
			continue
		}
		infos = append(infos, info)
		dom.Free()
	}

	return infos, nil
}

// Exists checks if a VM exists.
func (m *LibvirtManager) Exists(ctx context.Context, name string) (bool, error) {
	_, err := m.conn.LookupDomainByName(name)
	if err != nil {
		if isLibvirtError(err, libvirt.ERR_NO_DOMAIN) {
			return false, nil
		}
		return false, &VMError{Op: "exists", Err: err}
	}
	return true, nil
}

// Snapshot creates a VM snapshot.
func (m *LibvirtManager) Snapshot(ctx context.Context, name, snapshotName, description string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	xml := fmt.Sprintf(`<domainsnapshot>
  <name>%s</name>
  <description>%s</description>
</domainsnapshot>`, snapshotName, description)

	_, err = dom.CreateSnapshotXML(xml, 0)
	if err != nil {
		return &VMError{Op: "snapshot", Err: err}
	}

	return nil
}

// SnapshotList returns all snapshots for a VM.
func (m *LibvirtManager) SnapshotList(ctx context.Context, name string) ([]SnapshotInfo, error) {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	snapshots, err := dom.ListAllSnapshots(0)
	if err != nil {
		if isLibvirtError(err, libvirt.ERR_NO_DOMAIN) {
			return nil, nil
		}
		return nil, &VMError{Op: "snapshot_list", Err: err}
	}

	var infos []SnapshotInfo
	for i := range snapshots {
		info, err := m.snapshotToInfo(&snapshots[i])
		if err != nil {
			snapshots[i].Free()
			continue
		}
		infos = append(infos, info)
		snapshots[i].Free()
	}

	return infos, nil
}

// SnapshotRevert reverts to a snapshot.
func (m *LibvirtManager) SnapshotRevert(ctx context.Context, name, snapshotName string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	snap, err := dom.SnapshotLookupByName(snapshotName, 0)
	if err != nil {
		return &VMError{Op: "snapshot_revert", Err: fmt.Errorf("snapshot not found: %w", err)}
	}
	defer snap.Free()

	return snap.RevertToSnapshot(0)
}

// SnapshotDelete deletes a snapshot.
func (m *LibvirtManager) SnapshotDelete(ctx context.Context, name, snapshotName string) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	snap, err := dom.SnapshotLookupByName(snapshotName, 0)
	if err != nil {
		return &VMError{Op: "snapshot_delete", Err: fmt.Errorf("snapshot not found: %w", err)}
	}
	defer snap.Free()

	return snap.Delete(0)
}

// GetIP returns the VM's IP address.
func (m *LibvirtManager) GetIP(ctx context.Context, name string) (string, error) {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return "", err
	}
	defer dom.Free()

	// Get domain interfaces via QEMU agent or lease
	ifaces, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		// Try ARP table as fallback
		return m.getIPFromARP(dom)
	}

	for _, iface := range ifaces {
		if len(iface.Addrs) > 0 {
			return iface.Addrs[0].Addr, nil
		}
	}

	return "", fmt.Errorf("no IP address found")
}

// Shell starts an SSH shell on the VM.
func (m *LibvirtManager) Shell(ctx context.Context, name string) error {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return err
	}

	client, err := sshpkg.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	return client.Shell()
}

// Exec executes a command on the VM.
func (m *LibvirtManager) Exec(ctx context.Context, name string, command string) (stdout, stderr []byte, err error) {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	client, err := sshpkg.NewClient(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	return client.Exec(ctx, command)
}

// ExecInteractive executes a command with interactive terminal.
func (m *LibvirtManager) ExecInteractive(ctx context.Context, name string, command string) error {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return err
	}

	client, err := sshpkg.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// For interactive commands, we need a proper terminal
	// Use exec.Command to run ssh directly for better TTY support
	return execSSHCommand(cfg, command)
}

// Push copies a file to the VM.
func (m *LibvirtManager) Push(ctx context.Context, name string, localPath, remotePath string) error {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return err
	}

	client, err := sshpkg.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	transfer, err := sshpkg.NewTransfer(client)
	if err != nil {
		return fmt.Errorf("failed to create transfer client: %w", err)
	}
	defer transfer.Close()

	return transfer.Push(localPath, remotePath)
}

// Pull copies a file from the VM.
func (m *LibvirtManager) Pull(ctx context.Context, name string, remotePath, localPath string) error {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return err
	}

	client, err := sshpkg.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	transfer, err := sshpkg.NewTransfer(client)
	if err != nil {
		return fmt.Errorf("failed to create transfer client: %w", err)
	}
	defer transfer.Close()

	return transfer.Pull(remotePath, localPath)
}

// DeployKey deploys an SSH public key to the VM.
func (m *LibvirtManager) DeployKey(ctx context.Context, name string, publicKey []byte) error {
	cfg, err := m.getSSHConfig(ctx, name)
	if err != nil {
		return err
	}

	if m.keyManager == nil {
		return fmt.Errorf("key manager not initialized")
	}

	return m.keyManager.DeployKeyWithKey(ctx, name, cfg.Host, cfg.Port, cfg.User, cfg.KeyPath, publicKey)
}

// GenerateKeys generates SSH keys for VM access.
func (m *LibvirtManager) GenerateKeys(ctx context.Context, name string) (privateKey, publicKey []byte, err error) {
	if m.keyManager == nil {
		return nil, nil, fmt.Errorf("key manager not initialized")
	}

	info, err := m.keyManager.GenerateKey(name, "")
	if err != nil {
		return nil, nil, err
	}

	privateKey, err = os.ReadFile(info.PrivateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	publicKey, err = os.ReadFile(info.PublicKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read public key: %w", err)
	}

	return privateKey, publicKey, nil
}

// SetMemory changes VM memory allocation.
func (m *LibvirtManager) SetMemory(ctx context.Context, name string, memoryMB int) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	kb := uint64(memoryMB * 1024)
	return dom.SetMemory(kb)
}

// SetVCPUs changes VM CPU count.
func (m *LibvirtManager) SetVCPUs(ctx context.Context, name string, vcpus int) error {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return err
	}
	defer dom.Free()

	return dom.SetVcpusFlags(uint(vcpus), libvirt.DOMAIN_VCPU_LIVE|libvirt.DOMAIN_VCPU_CONFIG)
}

// GetConsole returns console access info.
func (m *LibvirtManager) GetConsole(ctx context.Context, name string) (string, error) {
	dom, err := m.lookupDomain(name)
	if err != nil {
		return "", err
	}
	defer dom.Free()

	// Return virsh console command
	// The actual console can be accessed via: virsh console <name>
	return fmt.Sprintf("virsh console %s", name), nil
}

// Internal methods

func (m *LibvirtManager) lookupDomain(name string) (*libvirt.Domain, error) {
	dom, err := m.conn.LookupDomainByName(name)
	if err != nil {
		if isLibvirtError(err, libvirt.ERR_NO_DOMAIN) {
			return nil, ErrNotFound
		}
		return nil, &VMError{Op: "lookup", Err: err}
	}
	return dom, nil
}

func (m *LibvirtManager) domainToInfo(dom *libvirt.Domain) (*VMInfo, error) {
	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}

	state, _, err := dom.GetState()
	if err != nil {
		return nil, err
	}

	info := &VMInfo{
		Name:      name,
		State:     m.libvirtStateToState(state),
		CreatedAt: time.Now(), // TODO: get actual creation time
	}

	// Get memory and vcpus
	domInfo, err := dom.GetInfo()
	if err == nil {
		info.Memory = int(domInfo.Memory) / 1024 // Convert from KB to MB
	}

	vcpus, err := dom.GetVcpusFlags(libvirt.DOMAIN_VCPU_LIVE)
	if err == nil {
		info.VCPUs = int(vcpus)
	}

	// Try to get IP
	ip, err := m.GetIP(context.Background(), name)
	if err == nil {
		info.IP = ip
	}

	return info, nil
}

func (m *LibvirtManager) snapshotToInfo(snap *libvirt.DomainSnapshot) (SnapshotInfo, error) {
	name, err := snap.GetName()
	if err != nil {
		return SnapshotInfo{}, err
	}

	isCurrent, err := snap.IsCurrent(0)
	if err != nil {
		isCurrent = false
	}

	// Get creation time from snapshot XML
	xml, err := snap.GetXMLDesc(0)
	var createdAt time.Time
	if err == nil {
		// Parse creation time from XML: <creationTime>unix_timestamp</creationTime>
		lines := strings.Split(xml, "\n")
		for _, line := range lines {
			if strings.Contains(line, "<creationTime>") {
				// Extract timestamp
				start := strings.Index(line, "<creationTime>") + len("<creationTime>")
				end := strings.Index(line, "</creationTime>")
				if start < end {
					tsStr := strings.TrimSpace(line[start:end])
					if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
						createdAt = time.Unix(ts, 0)
					}
				}
				break
			}
		}
	}

	return SnapshotInfo{
		Name:      name,
		CreatedAt: createdAt,
		Current:   isCurrent,
	}, nil
}

func (m *LibvirtManager) libvirtStateToState(state libvirt.DomainState) VMState {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return StateRunning
	case libvirt.DOMAIN_BLOCKED:
		return StateRunning
	case libvirt.DOMAIN_PAUSED:
		return StatePaused
	case libvirt.DOMAIN_SHUTDOWN:
		return StateStopped
	case libvirt.DOMAIN_SHUTOFF:
		return StateStopped
	case libvirt.DOMAIN_CRASHED:
		return StateCrashed
	case libvirt.DOMAIN_PMSUSPENDED:
		return StatePaused
	default:
		return StateUnknown
	}
}

func (m *LibvirtManager) generateDomainXML(cfg *VMConfig) (string, error) {
	memoryKB := cfg.Memory * 1024

	// Generate MAC if not provided
	mac := cfg.MAC
	if mac == "" {
		mac = generateMAC()
	}

	// Build disk path
	diskPath := cfg.Image
	if !filepath.IsAbs(diskPath) {
		// Relative to storage pool
		diskPath = filepath.Join("/var/lib/libvirt/images", cfg.Name+".qcow2")
	}

	xml := fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <memory unit='KiB'>%d</memory>
  <currentMemory unit='KiB'>%d</currentMemory>
  <vcpu placement='static'>%d</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
    <timer name='pit' tickpolicy='delay'/>
    <timer name='hpet' present='no'/>
  </clock>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <pm>
    <suspend-to-mem enabled='no'/>
    <suspend-to-disk enabled='no'/>
  </pm>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='network'>
      <mac address='%s'/>
      <source network='%s'/>
      <model type='virtio'/>
    </interface>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <graphics type='vnc' port='-1' autoport='yes' listen='127.0.0.1'>
      <listen type='address' address='127.0.0.1'/>
    </graphics>
  </devices>
</domain>`,
		cfg.Name,
		memoryKB,
		memoryKB,
		cfg.VCPUs,
		diskPath,
		mac,
		cfg.Network,
	)

	return xml, nil
}

func (m *LibvirtManager) getIPFromARP(dom *libvirt.Domain) (string, error) {
	// Get MAC address
	ifaces, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
	if err != nil {
		// Fallback: parse ARP table
		// TODO: implement ARP lookup
		return "", fmt.Errorf("no IP address available")
	}

	for _, iface := range ifaces {
		if len(iface.Addrs) > 0 {
			return iface.Addrs[0].Addr, nil
		}
	}

	return "", fmt.Errorf("no IP address found")
}

// Helper functions

func isLibvirtError(err error, errorCode libvirt.ErrorNumber) bool {
	if virErr, ok := err.(libvirt.Error); ok {
		return virErr.Code == errorCode
	}
	return false
}

func generateMAC() string {
	// Generate a random MAC with QEMU vendor prefix
	// 52:54:00 is QEMU's prefix
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x",
		uint8(time.Now().UnixNano()&0xff),
		uint8((time.Now().UnixNano()>>8)&0xff),
		uint8((time.Now().UnixNano()>>16)&0xff),
	)
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/root"
	}
	return home
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Validate VM name
func isValidVMName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// CloneVM creates a copy of an existing VM
func (m *LibvirtManager) CloneVM(ctx context.Context, sourceName, destName string, cfg *VMConfig) (*VMInfo, error) {
	// Check source exists
	exists, err := m.Exists(ctx, sourceName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	// Get source domain
	dom, err := m.lookupDomain(sourceName)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	// Get source XML
	xml, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, &VMError{Op: "clone", Err: err}
	}

	// Clone disk image
	// TODO: implement disk cloning

	// Create new domain with modified XML
	newXML := strings.ReplaceAll(xml, sourceName, destName)

	newDom, err := m.conn.DomainDefineXML(newXML)
	if err != nil {
		return nil, &VMError{Op: "clone", Err: fmt.Errorf("failed to define cloned domain: %w", err)}
	}
	defer newDom.Free()

	return m.Status(ctx, destName)
}

// getSSHConfig returns SSH configuration for a VM.
func (m *LibvirtManager) getSSHConfig(ctx context.Context, name string) (*sshpkg.Config, error) {
	info, err := m.Status(ctx, name)
	if err != nil {
		return nil, err
	}

	if info.State != StateRunning {
		return nil, ErrNotRunning
	}

	if info.IP == "" {
		return nil, fmt.Errorf("VM has no IP address")
	}

	// Get VM-specific SSH key
	keyPath, err := GetVMKeyPath(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get key path: %w", err)
	}

	// Check if key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Fall back to default key
		keyPath = ""
	}

	return &sshpkg.Config{
		Host:    info.IP,
		Port:    22,
		User:    "root",
		KeyPath: keyPath,
		Timeout: 30,
	}, nil
}

// execSSHCommand runs an SSH command with proper TTY handling.
func execSSHCommand(cfg *sshpkg.Config, command string) error {
	// Build ssh command arguments
	args := []string{}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	args = append(args, "-o", "StrictHostKeyChecking=accept-new")
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host))
	if command != "" {
		args = append(args, command)
	}

	// Run ssh command with TTY
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}