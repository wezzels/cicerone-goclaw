# Cicerone Deploy Command Guide

Complete guide for VM workspace management with Cicerone.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Command Reference](#command-reference)
4. [Examples](#examples)
5. [Troubleshooting](#troubleshooting)
6. [Architecture](#architecture)

---

## Prerequisites

### Install libvirt

```bash
# Ubuntu/Debian
sudo apt-get install -y libvirt-daemon-system libvirt-clients qemu-kvm

# Add user to libvirt group
sudo usermod -aG libvirt $(whoami)

# Logout and login again for group changes to take effect
```

### Build Cicerone with libvirt support

```bash
# Clone repository
git clone https://github.com/wezzels/cicerone-goclaw.git
cd cicerone-goclaw

# Build with libvirt support
CGO_ENABLED=1 go build -tags libvirt -o cicerone .

# Verify
./cicerone deploy list
```

### Configuration

Create `~/.cicerone/config.yaml`:

```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b
  timeout: 60

workspace:
  path: ~/.cicerone/workspace

vms:
  dev:
    name: my-dev-vm
    image: /var/lib/libvirt/images/my-vm.qcow2
    memory: 4096
    vcpus: 2
    network: default
    user: root
    ssh_key: ~/.cicerone/keys/my-dev-vm
```

---

## Quick Start

### 1. List Available VMs

```bash
./cicerone deploy list
```

Output:
```
VM Workspaces
=============

NAME      STATE    IP               MEMORY   VCPUS
debian11  running  192.168.122.164  8192MB   4
fedora41  running  10.0.20.156      16384MB  4
kali      stopped                    4096MB   0
```

### 2. Check VM Status

```bash
./cicerone deploy status debian11
```

Output:
```
VM: debian11
State: running
IP: 192.168.122.164
Memory: 8192 MB
vCPUs: 4
Created: 2026-04-10 18:36:15
```

### 3. Generate and Deploy SSH Keys

```bash
# Generate SSH key for VM
./cicerone deploy keys debian11 --generate

# Deploy key to VM (requires password access initially)
./cicerone deploy keys debian11 --deploy
# Enter root password when prompted

# Verify key status
./cicerone deploy keys debian11 --status
```

### 4. Execute Commands

```bash
# Run a command
./cicerone deploy exec debian11 "uname -a"
# Linux debian11 6.1.0-18-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.76-1 (2024-02-01) x86_64 GNU/Linux

# Open interactive shell
./cicerone deploy shell debian11
```

### 5. Transfer Files

```bash
# Push file to VM
./cicerone deploy push debian11 ./local-file.txt /workspace/remote-file.txt

# Pull file from VM
./cicerone deploy pull debian11 /workspace/remote-file.txt ./local-file.txt
```

### 6. Manage Snapshots

```bash
# Create snapshot before changes
./cicerone deploy snapshot debian11 --create --name "before-tests"

# List snapshots
./cicerone deploy snapshot debian11 --list

# Revert to snapshot
./cicerone deploy snapshot debian11 --revert --name "before-tests"

# Delete snapshot
./cicerone deploy snapshot debian11 --delete --name "before-tests"
```

---

## Command Reference

### `cicerone deploy list`

List all configured and running VMs.

**Usage:**
```bash
cicerone deploy list [flags]
```

**Flags:**
- `-h, --help` - Help for list

**Output:**
- NAME - VM name
- STATE - running, stopped, paused, etc.
- IP - Current IP address (if running)
- MEMORY - Allocated memory
- VCPUS - Number of virtual CPUs

**Example:**
```bash
$ cicerone deploy list
VM Workspaces
=============

NAME            STATE    IP               MEMORY   VCPUS
debian11        running  192.168.122.164  8192MB   4
fedora41        running  10.0.20.156      16384MB  4
cicerone-fresh  running                   4096MB   2
```

---

### `cicerone deploy status`

Show detailed VM status.

**Usage:**
```bash
cicerone deploy status <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `-h, --help` - Help for status

**Example:**
```bash
$ cicerone deploy status debian11
VM: debian11
State: running
IP: 192.168.122.164
Memory: 8192 MB
vCPUs: 4
Created: 2026-04-10 18:36:15
```

---

### `cicerone deploy create`

Create a new VM from a base image.

**Usage:**
```bash
cicerone deploy create <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `--autostart` - Start VM after creation
- `-i, --image string` - Path to base image (required)
- `-m, --memory int` - Memory in MB (default 2048)
- `-n, --network string` - Network name (default "default")
- `-c, --vcpus int` - Number of vCPUs (default 2)
- `-h, --help` - Help for create

**Example:**
```bash
# Create VM from image
$ cicerone deploy create dev-vm --image /var/lib/libvirt/images/base.qcow2 --memory 4096 --vcpus 2

# Create and start
$ cicerone deploy create dev-vm --image /var/lib/libvirt/images/base.qcow2 --autostart
```

---

### `cicerone deploy start`

Start a stopped VM.

**Usage:**
```bash
cicerone deploy start <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `-h, --help` - Help for start

**Example:**
```bash
$ cicerone deploy start debian11
✓ Started VM 'debian11'
```

---

### `cicerone deploy stop`

Stop a running VM.

**Usage:**
```bash
cicerone deploy stop <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `-f, --force` - Force stop (destroy) - like pulling power cable
- `-h, --help` - Help for stop

**Example:**
```bash
# Graceful shutdown
$ cicerone deploy stop debian11

# Force stop (immediate)
$ cicerone deploy stop debian11 --force
```

---

### `cicerone deploy restart`

Restart a VM.

**Usage:**
```bash
cicerone deploy restart <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `-h, --help` - Help for restart

**Example:**
```bash
$ cicerone deploy restart debian11
✓ Restarted VM 'debian11'
```

---

### `cicerone deploy shell`

Open an interactive SSH shell on the VM.

**Usage:**
```bash
cicerone deploy shell <name> [flags]
```

**Prerequisites:**
- VM must be running
- SSH key must be deployed (use `cicerone deploy keys --deploy`)

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `-h, --help` - Help for shell

**Example:**
```bash
$ cicerone deploy shell debian11
root@debian11:~# hostname
debian11
root@debian11:~# exit
```

---

### `cicerone deploy exec`

Execute a command on the VM via SSH.

**Usage:**
```bash
cicerone deploy exec <name> <command> [flags]
```

**Arguments:**
- `<name>` - VM name (required)
- `<command>` - Command to execute (required)

**Flags:**
- `-h, --help` - Help for exec

**Example:**
```bash
$ cicerone deploy exec debian11 "uname -a"
Linux debian11 6.1.0-18-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.76-1 x86_64 GNU/Linux

$ cicerone deploy exec debian11 "ls -la /workspace"
total 8
drwxr-xr-x 2 root root 4096 Apr 10 18:00 .
drwxr-xr-x 3 root root 4096 Apr 10 17:55 ..
```

---

### `cicerone deploy push`

Push a file to the VM via SFTP.

**Usage:**
```bash
cicerone deploy push <name> <local-path> <remote-path> [flags]
```

**Arguments:**
- `<name>` - VM name (required)
- `<local-path>` - Local file path (required)
- `<remote-path>` - Remote file path on VM (required)

**Flags:**
- `-h, --help` - Help for push

**Example:**
```bash
$ cicerone deploy push debian11 ./my-script.sh /workspace/my-script.sh
✓ Pushed ./my-script.sh to /workspace/my-script.sh

$ cicerone deploy exec debian11 "chmod +x /workspace/my-script.sh"
```

---

### `cicerone deploy pull`

Pull a file from the VM via SFTP.

**Usage:**
```bash
cicerone deploy pull <name> <remote-path> <local-path> [flags]
```

**Arguments:**
- `<name>` - VM name (required)
- `<remote-path>` - Remote file path on VM (required)
- `<local-path>` - Local file path (required)

**Flags:**
- `-h, --help` - Help for pull

**Example:**
```bash
$ cicerone deploy pull debian11 /var/log/syslog ./syslog.log
✓ Pulled /var/log/syslog to ./syslog.log
```

---

### `cicerone deploy keys`

Manage SSH keys for VM access.

**Usage:**
```bash
cicerone deploy keys <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `--generate` - Generate new SSH key pair
- `--deploy` - Deploy key to VM (requires password)
- `--status` - Show key status
- `--key string` - Path to existing SSH key to use
- `-h, --help` - Help for keys

**Examples:**
```bash
# Generate new key
$ cicerone deploy keys debian11 --generate
✓ Generated SSH key for VM 'debian11'
  Private key: /home/user/.cicerone/keys/id_ed25519_debian11
  Public key:  /home/user/.cicerone/keys/id_ed25519_debian11.pub

# Deploy key to VM
$ cicerone deploy keys debian11 --deploy
Enter root password for debian11:
✓ SSH key deployed to debian11

# Check status
$ cicerone deploy keys debian11 --status
SSH key configured for VM 'debian11'
  Private key: /home/user/.cicerone/keys/id_ed25519_debian11
  Public key:  /home/user/.cicerone/keys/id_ed25519_debian11.pub

# Use existing key
$ cicerone deploy keys debian11 --key ~/.ssh/id_ed25519
✓ SSH key configured for VM 'debian11'
```

---

### `cicerone deploy snapshot`

Manage VM snapshots.

**Usage:**
```bash
cicerone deploy snapshot <name> [flags]
```

**Arguments:**
- `<name>` - VM name (required)

**Flags:**
- `--create` - Create snapshot
- `--delete` - Delete snapshot
- `--description string` - Snapshot description
- `--list` - List snapshots
- `--name string` - Snapshot name (required for create/revert/delete)
- `--revert` - Revert to snapshot
- `-h, --help` - Help for snapshot

**Examples:**
```bash
# Create snapshot
$ cicerone deploy snapshot debian11 --create --name "before-tests" --description "Before running test suite"
✓ Snapshot 'before-tests' created

# List snapshots
$ cicerone deploy snapshot debian11 --list
Snapshots for VM 'debian11':
NAME             CREATED           CURRENT
before-tests     2026-04-10 17:33  
after-setup      2026-04-10 17:35  *

# Revert to snapshot
$ cicerone deploy snapshot debian11 --revert --name "before-tests"
✓ Reverted to snapshot 'before-tests'

# Delete snapshot
$ cicerone deploy snapshot debian11 --delete --name "before-tests"
✓ Snapshot 'before-tests' deleted
```

---

### `cicerone deploy workspace`

Set active VM workspace for agent execution.

**Usage:**
```bash
cicerone deploy workspace [name] [flags]
```

**Arguments:**
- `[name]` - VM name to activate, or empty to deactivate

**Flags:**
- `-h, --help` - Help for workspace

**Examples:**
```bash
# Set active workspace
$ cicerone deploy workspace debian11
✓ Active workspace set to: debian11

# Show current workspace
$ cicerone deploy workspace
Active workspace: debian11

# Deactivate workspace (switch to local)
$ cicerone deploy workspace local
✓ Workspace deactivated. Using local execution.
```

---

## Examples

### Development Workflow

```bash
# 1. List available VMs
$ cicerone deploy list

# 2. Start VM
$ cicerone deploy start debian11

# 3. Setup SSH keys
$ cicerone deploy keys debian11 --generate
$ cicerone deploy keys debian11 --deploy

# 4. Create a snapshot before changes
$ cicerone deploy snapshot debian11 --create --name "clean-state"

# 5. Set active workspace for agent
$ cicerone deploy workspace debian11

# 6. Run commands
$ cicerone deploy exec debian11 "git clone https://github.com/example/repo.git"
$ cicerone deploy exec debian11 "cd repo && make build"

# 7. If something breaks, revert
$ cicerone deploy snapshot debian11 --revert --name "clean-state"

# 8. When done, deactivate workspace
$ cicerone deploy workspace local

# 9. Stop VM
$ cicerone deploy stop debian11
```

### Push and Run Script

```bash
# Push script
$ cicerone deploy push debian11 ./deploy.sh /workspace/deploy.sh

# Make executable
$ cicerone deploy exec debian11 "chmod +x /workspace/deploy.sh"

# Run script
$ cicerone deploy exec debian11 "/workspace/deploy.sh"
```

### File Transfer Workflow

```bash
# Push configuration
$ cicerone deploy push debian11 ./config.yaml /workspace/config.yaml

# Run with config
$ cicerone deploy exec debian11 "myapp --config /workspace/config.yaml"

# Pull results
$ cicerone deploy pull debian11 /workspace/output.log ./output.log
```

---

## Troubleshooting

### VM Not Listed

**Problem:** VM doesn't appear in `deploy list`

**Solution:**
1. Check libvirt is running: `sudo systemctl status libvirtd`
2. Check VM exists: `sudo virsh list --all`
3. Add VM to config.yaml:

```yaml
vms:
  my-vm:
    name: my-vm
    image: /var/lib/libvirt/images/my-vm.qcow2
    memory: 4096
    vcpus: 2
    network: default
```

### SSH Connection Failed

**Problem:** `Permission denied (publickey,password)`

**Solution:**
1. Generate key: `cicerone deploy keys <vm> --generate`
2. Deploy key: `cicerone deploy keys <vm> --deploy`
3. Enter root password when prompted
4. If still failing, check VM allows root login:
   ```bash
   ssh root@<vm-ip>
   # Password: <root-password>
   
   # Enable root login
   sudo sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
   sudo systemctl restart sshd
   ```

### VM Won't Start

**Problem:** `cicerone deploy start` fails

**Solution:**
1. Check error: `sudo virsh start <vm>`
2. Check VM XML: `sudo virsh dumpxml <vm>`
3. Check logs: `sudo journalctl -u libvirtd`
4. Verify disk exists: `ls -l /var/lib/libvirt/images/<vm>.qcow2`

### Snapshot Fails

**Problem:** `cicerone deploy snapshot --create` fails

**Solution:**
1. Check VM is running: `cicerone deploy status <vm>`
2. Check libvirt supports snapshots: `sudo virsh snapshot-list <vm>`
3. For external snapshots, check disk format: `qemu-img info <disk>.qcow2`

### IP Address Not Shown

**Problem:** VM shows no IP address

**Solution:**
1. Check VM is running: `cicerone deploy status <vm>`
2. Check VM network: `sudo virsh net-list`
3. Get IP manually: `sudo virsh domifaddr <vm>`
4. If empty, VM may not have gotten DHCP lease yet - wait a few seconds

### No IP Available Error

**Problem:** `no IP address available`

**Cause:** VM guest agent not responding

**Solution:**
1. Install qemu-guest-agent in VM:
   ```bash
   cicerone deploy exec <vm> "apt-get install -y qemu-guest-agent"
   cicerone deploy restart <vm>
   ```

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Cicerone CLI                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────┐   │
│  │  deploy.go   │────▶│  vm/vm.go   │────▶│  vm/libvirt.go     │   │
│  │  (commands) │     │  (interface)│     │  (implementation)   │   │
│  └─────────────┘     └─────────────┘     └─────────────────────┘   │
│                                                 │                   │
│                                                 ▼                   │
│                                ┌────────────────────────────────┐  │
│                                │     libvirt.org/go/libvirt     │  │
│                                │         (Go bindings)          │  │
│                                └────────────────────────────────┘  │
│                                                 │                   │
│                                                 ▼                   │
│                                ┌────────────────────────────────┐  │
│                                │      libvirtd (QEMU/KVM)       │  │
│                                └────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### File Structure

```
cmd/deploy.go              - CLI commands
internal/vm/
  vm.go                    - Manager interface, types
  libvirt.go               - LibvirtManager implementation
  libvirt_stub.go          - Stub for non-libvirt builds
  config.go                - VM configuration loading
  keys.go                  - SSH key management
internal/ssh/
  client.go                - SSH client wrapper
```

### Build Tags

```bash
# Without libvirt (stub implementation)
go build ./...

# With libvirt (full implementation)
CGO_ENABLED=1 go build -tags libvirt ./...
```

### Known Limitations (TODOs)

1. **VM Creation Time** - Status shows `CreatedAt` as current time, not actual VM creation
2. **ARP Lookup** - IP discovery uses guest agent, ARP fallback not implemented
3. **Disk Cloning** - `Clone()` creates domain but doesn't clone disk image

---

## Configuration Reference

### ~/.cicerone/config.yaml

```yaml
# LLM Configuration
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.1:8b
  timeout: 60

# Local workspace
workspace:
  path: ~/.cicerone/workspace

# VM configurations
vms:
  # Development VM
  dev:
    name: dev-vm
    image: /var/lib/libvirt/images/dev-vm.qcow2
    memory: 4096        # MB
    vcpus: 2
    network: default    # libvirt network name
    user: root
    ssh_key: ~/.cicerone/keys/dev-vm
    auto_start: false
    
  # Test VM
  test:
    name: test-vm
    image: /var/lib/libvirt/images/test-vm.qcow2
    memory: 2048
    vcpus: 1
    network: default
    user: root
    ssh_key: ~/.cicerone/keys/test-vm

# Deploy defaults
deploy:
  default_vm: dev
  default_network: default
  storage_pool: default
```

---

## Appendix: Libvirt Commands Reference

For operations not covered by Cicerone:

```bash
# List all VMs
sudo virsh list --all

# Start VM
sudo virsh start <vm>

# Stop VM (graceful)
sudo virsh shutdown <vm>

# Stop VM (force)
sudo virsh destroy <vm>

# Get VM IP
sudo virsh domifaddr <vm>

# Console access
sudo virsh console <vm>

# VNC display
sudo virsh vncdisplay <vm>

# VM info
sudo virsh dominfo <vm>

# Disk info
sudo qemu-img info /var/lib/libvirt/images/<vm>.qcow2

# Create snapshot
sudo virsh snapshot-create-as <vm> "snapshot-name" "description"

# List snapshots
sudo virsh snapshot-list <vm>

# Revert snapshot
sudo virsh snapshot-revert <vm> "snapshot-name"

# Delete snapshot
sudo virsh snapshot-delete <vm> "snapshot-name"
```