# Cicerone Deploy - VM Workspace Management

## Overview

The `cicerone deploy` command manages VM-based workspaces for isolated development and execution. It integrates with libvirt/QEMU to create, configure, and manage VMs that serve as remote workspaces.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Cicerone Host                                │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────────┐│
│  │   Config    │────▶│   Deploy    │────▶│    SSH Client           ││
│  │ ~/.cicerone │     │   Manager   │     │ internal/ssh            ││
│  └─────────────┘     └──────┬──────┘     └─────────────────────────┘│
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                      VM Workspace                                ││
│  │  ┌───────────────┐  ┌────────────┐  ┌─────────────────────────┐ ││
│  │  │ libvirt/QEMU  │  │   NAT      │  │   SSH Root Access      │ ││
│  │  │   wezzelos    │  │   Network  │  │   via SSH keys         │ ││
│  │  └───────────────┘  └────────────┘  └─────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## Configuration

### ~/.cicerone/config.yaml

```yaml
# LLM Configuration
llm:
  base_url: http://localhost:11434
  model: llama3.1:8b
  provider: ollama
  timeout: 300

# Local workspace (default)
workspace:
  path: /home/user/.cicerone/workspace

# VM workspaces (optional)
vms:
  # Development VM
  dev:
    name: wezzelos-dev
    image: /var/lib/libvirt/images/wezzelos-base.qcow2
    memory: 4096
    vcpus: 2
    network: default
    ip: 192.168.122.100
    ssh_key: ~/.ssh/id_ed25519
    user: root
    description: "Development workspace VM"
  
  # Production VM
  prod:
    name: wezzelos-prod
    image: /var/lib/libvirt/images/wezzelos-base.qcow2
    memory: 8192
    vcpus: 4
    network: default
    ip: 192.168.122.101
    ssh_key: ~/.ssh/id_ed25519
    user: root
    description: "Production workspace VM"

# Deployment settings
deploy:
  default_vm: dev
  auto_start: true
  snapshot_on_stop: false
```

## Commands

### List VMs

```bash
cicerone deploy list

# Output:
# VM Workspaces
# =============
# 
#   dev (wezzelos-dev)
#     Status: running
#     IP: 192.168.122.100
#     Memory: 4GB
#     CPUs: 2
#     Workspace: /root/workspace
# 
#   prod (wezzelos-prod)
#     Status: stopped
#     IP: 192.168.122.101
#     Memory: 8GB
#     CPUs: 4
```

### Create VM

```bash
# Create from base image
cicerone deploy create dev

# Create with custom settings
cicerone deploy create dev --memory 8192 --vcpus 4

# Create from scratch
cicerone deploy create dev --image ubuntu-22.04.qcow2 --fresh
```

### Start/Stop VM

```bash
cicerone deploy start dev
cicerone deploy stop dev
cicerone deploy restart dev
```

### Connect to VM

```bash
# SSH shell
cicerone deploy shell dev

# Execute command
cicerone deploy exec dev "ls -la /workspace"

# Push files
cicerone deploy push dev ./local.txt /workspace/remote.txt

# Pull files
cicerone deploy pull dev /workspace/remote.txt ./local.txt
```

### Workspace Operations

```bash
# Set VM as active workspace
cicerone deploy workspace dev

# Now /task commands run on VM
/task Create a flask app in /workspace

# Switch back to local
cicerone deploy workspace local
```

### SSH Key Management

```bash
# Generate and deploy SSH keys
cicerone deploy keys dev --generate

# Use existing key
cicerone deploy keys dev --key ~/.ssh/id_ed25519

# Show key status
cicerone deploy keys dev --status
```

### Snapshot Management

```bash
# Create snapshot
cicerone deploy snapshot dev --create "before-tests"

# List snapshots
cicerone deploy snapshot dev --list

# Revert to snapshot
cicerone deploy snapshot dev --revert "before-tests"
```

## Implementation

### Command Structure

```
cmd/deploy.go
├── deployCmd          # Parent command
├── deployListCmd      # List VMs
├── deployCreateCmd    # Create VM
├── deployStartCmd     # Start VM
├── deployStopCmd      # Stop VM
├── deployShellCmd     # SSH shell
├── deployExecCmd      # Execute command
├── deployPushCmd      # Push files
├── deployPullCmd      # Pull files
├── deployKeysCmd      # Key management
├── deploySnapshotCmd  # Snapshots
└── deployWorkspaceCmd # Set active workspace
```

### Internal Package

```
internal/vm/
├── vm.go           # VM management interface
├── libvirt.go      # libvirt implementation
├── config.go       # VM configuration
├── keys.go         # SSH key management
└── snapshot.go     # Snapshot management
```

### Integration Points

1. **Agent Integration**: When `deploy workspace <vm>` is set, the agent's `run_shell` and `write_file` operations execute remotely via SSH.

2. **Config Integration**: VM settings stored in `~/.cicerone/config.yaml` under `vms` key.

3. **SSH Integration**: Uses existing `internal/ssh` package for remote execution.

## SSH Key Setup for Root Access

The `deploy keys` command handles:

1. **Key Generation**: Creates Ed25519 keys if needed
2. **Key Deployment**: Copies public key to VM's `/root/.ssh/authorized_keys`
3. **Known Hosts**: Adds VM to `~/.ssh/known_hosts`
4. **Config Update**: Updates `~/.ssh/config` for easy access

```bash
# Full key setup
cicerone deploy keys dev --setup

# This generates:
# ~/.ssh/id_ed25519_cicerone_dev (private key)
# ~/.ssh/id_ed25519_cicerone_dev.pub (public key)
# And deploys to:
# dev:/root/.ssh/authorized_keys
```

## NAT Networking

VMs use NAT networking (libvirt default network):

```
Host (192.168.122.1)
  │
  ├── VM dev (192.168.122.100)
  │     └── Port forwards: None (SSH via libvirt network)
  │
  └── VM prod (192.168.122.101)
        └── Port forwards: None (SSH via libvirt network)
```

Access via SSH:
```bash
ssh root@192.168.122.100
# Or using SSH config:
ssh wezzelos-dev
```

## Example Workflow

```bash
# 1. Create development VM
cicerone deploy create dev

# 2. Start VM
cicerone deploy start dev

# 3. Setup SSH keys
cicerone deploy keys dev --setup

# 4. Set as active workspace
cicerone deploy workspace dev

# 5. Run tasks on VM
/task Create a flask app with Dockerfile

# 6. Snapshot before risky changes
cicerone deploy snapshot dev --create "before-experiment"

# 7. Push local files
cicerone deploy push dev ./src/ /workspace/src/

# 8. Switch back to local when done
cicerone deploy workspace local

# 9. Stop VM
cicerone deploy stop dev
```

## Security Considerations

1. **Root Access**: VMs are designed for development with root access
2. **NAT Isolation**: VMs are not directly accessible from outside
3. **Key-based Auth**: Password authentication is disabled
4. **Snapshot Safety**: Create snapshots before dangerous operations

## Error Handling

- VM not found: `Error: VM 'dev' not found. Run 'cicerone deploy list' to see available VMs.`
- VM not running: `Error: VM 'dev' is not running. Run 'cicerone deploy start dev'.`
- SSH connection failed: `Error: Cannot connect to VM 'dev'. Check if SSH keys are deployed.`
- Insufficient resources: `Error: Not enough memory to start VM 'dev'. Need 4GB, available 2GB.`