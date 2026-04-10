# Cicerone Deploy - Implementation Status

## Fully Implemented Commands

| Command | Status | Notes |
|---------|--------|-------|
| `deploy list` | ✅ Complete | Lists all VMs with status |
| `deploy status` | ✅ Complete | Shows detailed VM info |
| `deploy create` | ✅ Complete | Creates VM from image |
| `deploy start` | ✅ Complete | Starts VM |
| `deploy stop` | ✅ Complete | Stops VM (graceful or force) |
| `deploy restart` | ✅ Complete | Restarts VM |
| `deploy snapshot --create` | ✅ Complete | Creates snapshot |
| `deploy snapshot --list` | ✅ Complete | Lists snapshots |
| `deploy snapshot --revert` | ✅ Complete | Reverts to snapshot |
| `deploy snapshot --delete` | ✅ Complete | Deletes snapshot |
| `deploy shell` | ✅ Complete | Interactive SSH shell |
| `deploy exec` | ✅ Complete | Execute command via SSH |
| `deploy push` | ✅ Complete | Push file via SFTP |
| `deploy pull` | ✅ Complete | Pull file via SFTP |
| `deploy keys --generate` | ✅ Complete | Generate SSH key pair |
| `deploy keys --deploy` | ✅ Complete | Deploy key to VM |
| `deploy keys --status` | ✅ Complete | Show key status |
| `deploy workspace` | ✅ Complete | Set active workspace |

## Test Results

### Working Commands

```bash
# List VMs - WORKS
$ ./cicerone-goclaw deploy list
NAME            STATE    IP               MEMORY   VCPUS
debian11        running  192.168.122.164  8192MB   4
fedora41        running  10.0.20.156      16384MB  4

# Status - WORKS
$ ./cicerone-goclaw deploy status debian11
VM: debian11
State: running
IP: 192.168.122.164
Memory: 8192 MB
vCPUs: 4

# Keys generate - WORKS
$ ./cicerone-goclaw deploy keys debian11 --generate
✓ Generated SSH key for VM 'debian11'
  Private key: /home/user/.cicerone/keys/id_ed25519_debian11
  Public key:  /home/user/.cicerone/keys/id_ed25519_debian11.pub

# Keys status - WORKS
$ ./cicerone-goclaw deploy keys debian11 --status
SSH key configured for VM 'debian11'

# Snapshot create - WORKS
$ ./cicerone-goclaw deploy snapshot debian11 --create --name "test"
✓ Snapshot 'test' created

# Snapshot list - WORKS
$ ./cicerone-goclaw deploy snapshot debian11 --list
NAME             CREATED           CURRENT
test             2026-04-10 17:33  *

# Snapshot revert - WORKS
$ ./cicerone-goclaw deploy snapshot debian11 --revert --name "test"
✓ Reverted to snapshot 'test'

# Snapshot delete - WORKS
$ ./cicerone-goclaw deploy snapshot debian11 --delete --name "test"
✓ Snapshot 'test' deleted
```

### Commands Requiring VM with SSH Key

```bash
# shell - Requires SSH key deployed
$ ./cicerone-goclaw deploy shell debian11
# Opens interactive SSH shell

# exec - Requires SSH key deployed
$ ./cicerone-goclaw deploy exec debian11 "uname -a"
# Returns command output

# push/pull - Requires SSH key deployed
$ ./cicerone-goclaw deploy push debian11 ./file.txt /workspace/file.txt
$ ./cicerone-goclaw deploy pull debian11 /workspace/file.txt ./file.txt
```

## Known Limitations (TODOs)

### 1. VM Creation Time Not Accurate
**Location:** `internal/vm/libvirt.go:558`

```go
CreatedAt: time.Now(), // TODO: get actual creation time
```

**Impact:** Status shows current time instead of actual VM creation time.

**Fix Required:**
- Query libvirt for domain creation time
- Store in VM metadata or cache

**Priority:** Low (cosmetic issue)

---

### 2. IP Discovery - ARP Fallback Not Implemented
**Location:** `internal/vm/libvirt.go:720`

```go
// TODO: implement ARP lookup
return "", fmt.Errorf("no IP address available")
```

**Impact:** IP discovery relies on qemu-guest-agent. If agent not installed, IP won't be detected.

**Workaround:**
1. Install qemu-guest-agent in VM
2. Use `virsh domifaddr <vm>` manually

**Fix Required:**
- Parse ARP table: `arp -an | grep <mac>`
- Or parse DHCP leases: `/var/lib/libvirt/dnsmasq/default.leases`
- Or use `nmap` for network scan

**Priority:** Medium (convenience)

---

### 3. Clone Not Implemented
**Location:** `internal/vm/libvirt.go:803`

```go
// Clone disk image
// TODO: implement disk cloning
```

**Impact:** `Clone()` creates domain XML but doesn't clone the disk.

**Workaround:**
```bash
# Clone VM manually
sudo qemu-img create -f qcow2 -b /var/lib/libvirt/images/source.qcow2 /var/lib/libvirt/images/clone.qcow2
sudo virsh define clone.xml
```

**Fix Required:**
- Copy backing disk
- Update XML with new disk path
- Optionally: use `virsh clone` command

**Priority:** Low (use `create` from existing image)

---

### 4. SSH Host Key Verification Disabled
**Location:** `internal/ssh/client.go:56`

```go
HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
```

**Impact:** SSH connections accept any host key (MITM vulnerability).

**Security Risk:** Medium - man-in-the-middle attacks possible

**Fix Required:**
- Add host key to known_hosts on first connection
- Verify host key on subsequent connections
- Option to accept new host keys

**Priority:** Medium (security)

---

### 5. Libvirt Stub for Non-libvirt Builds
**Location:** `internal/vm/libvirt_stub.go`

```go
// LibvirtManager is a stub when libvirt is not available.
type LibvirtManager struct{}

func NewLibvirtManager(opts *ManagerOptions) (*LibvirtManager, error) {
    return nil, ErrLibvirtNotAvailable
}
```

**Impact:** When built without `-tags libvirt`, deploy commands return error.

**Expected Behavior:** This is by design - libvirt is optional.

**Workaround:**
```bash
# Build with libvirt support
CGO_ENABLED=1 go build -tags libvirt -o cicerone .
```

**Priority:** N/A (by design)

---

## Architecture Notes

### Build Tags

| Tag | Files Included | Behavior |
|-----|---------------|----------|
| (none) | `libvirt_stub.go` | Stub implementation, returns errors |
| `libvirt` | `libvirt.go` | Full implementation with libvirt bindings |

### Dependencies

**With libvirt:**
```
libvirt.org/go/libvirt
libvirt-dev (system package)
```

**Without libvirt:**
- No external dependencies
- Deploy commands return error

### SSH Integration

```
cmd/deploy.go (commands)
       │
       ▼
internal/vm/libvirt.go (GetIP, Shell, Exec, Push, Pull)
       │
       ▼
internal/ssh/client.go (SSHClient, SFTP)
       │
       ▼
golang.org/x/crypto/ssh
```

---

## Testing Checklist

### Environment Setup
- [x] libvirt installed and running
- [x] VM created from ISO or existing image
- [x] VM has SSH server installed
- [x] VM has qemu-guest-agent (for IP discovery)

### Command Tests
- [x] `deploy list` - Lists VMs correctly
- [x] `deploy status` - Shows VM details
- [x] `deploy create` - Creates VM from image
- [x] `deploy start` - Starts stopped VM
- [x] `deploy stop` - Stops running VM
- [x] `deploy restart` - Restarts VM
- [x] `deploy keys --generate` - Creates SSH keys
- [x] `deploy keys --status` - Shows key status
- [x] `deploy keys --deploy` - Deploys key (requires password)
- [ ] `deploy shell` - Opens interactive shell (requires key deployed)
- [ ] `deploy exec` - Runs command (requires key deployed)
- [ ] `deploy push` - Copies file to VM (requires key deployed)
- [ ] `deploy pull` - Copies file from VM (requires key deployed)
- [x] `deploy snapshot --create` - Creates snapshot
- [x] `deploy snapshot --list` - Lists snapshots
- [x] `deploy snapshot --revert` - Reverts snapshot
- [x] `deploy snapshot --delete` - Deletes snapshot
- [ ] `deploy workspace` - Sets active workspace (tested in code)

### Integration Tests
- [x] Test file: `internal/vm/vm_test.go`
- [x] Test file: `internal/vm/config_test.go`
- [x] Test file: `internal/vm/snapshot_test.go`
- [x] Test file: `internal/vm/keys_test.go` (exists but needs SSH server)
- [ ] Integration test with real VM

---

## Future Enhancements

### High Priority
1. SSH host key verification
2. ARP/DHCP lease parsing for IP discovery

### Medium Priority
3. Actual VM creation time
4. Progress bars for file transfer
5. Concurrent file transfers

### Low Priority
6. Disk cloning for `Clone()`
7. VM templates
8. Network configuration
9. Cloud-init support
10. VM export/import

---

## Quick Reference

### Build Commands

```bash
# Development (without libvirt)
go build -o cicerone ./cmd

# Production (with libvirt)
CGO_ENABLED=1 go build -tags libvirt -o cicerone ./cmd

# Test
go test -short ./...

# Test with libvirt
go test -tags libvirt -short ./...

# Lint
golangci-lint run --timeout 5m ./...
```

### VM Setup Commands

```bash
# Generate SSH key
./cicerone deploy keys dev --generate

# Deploy key (enter password)
./cicerone deploy keys dev --deploy

# Test connection
./cicerone deploy exec dev "hostname"

# Create snapshot before changes
./cicerone deploy snapshot dev --create --name "clean"

# Work...
./cicerone deploy exec dev "apt-get update"

# Revert if needed
./cicerone deploy snapshot dev --revert --name "clean"
```