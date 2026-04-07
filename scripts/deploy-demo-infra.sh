#!/bin/bash
# Cicerone Demo Infrastructure Deployment Script
# Builds complete infrastructure on demo_nat network using only cicerone

set -o pipefail

DEMO_DIR="$HOME/demo-infra"
DISK_DIR="$DEMO_DIR/disks"
CLOUD_DIR="$DEMO_DIR/cloud-init"
LOG_FILE="$DEMO_DIR/deployment.log"

# Initialize log
mkdir -p "$DEMO_DIR"
echo "=== Deployment started: $(date) ===" | tee "$LOG_FILE"

# VM definitions in order of deployment
# name:ip:vcpu:mem_gb:user:description
VM_DEFS=(
  "demo-admin1:192.168.200.10:4:8:steve:AI Admin 1"
  "demo-admin2:192.168.200.11:4:8:steve:AI Admin 2"
  "demo-siem:192.168.200.20:2:4:steve:SIEM Node"
  "demo-runner1:192.168.200.30:2:4:steve:GitLab Runner 1"
  "demo-runner2:192.168.200.31:2:4:steve:GitLab Runner 2"
  "demo-runner3:192.168.200.32:2:4:steve:GitLab Runner 3"
  "demo-runner4:192.168.200.33:2:4:steve:GitLab Runner 4"
  "demo-runner5:192.168.200.34:2:4:steve:GitLab Runner 5"
  "demo-gateway:192.168.200.40:2:4:steve:Tailscale Gateway"
  "demo-email:192.168.200.50:1:2:steve:Email Server"
)

DEFAULT_PASS="Demo2024!"

deploy_vm() {
  local name="$1"
  local ip="$2"
  local vcpu="$3"
  local mem="$4"
  local user="$5"
  
  echo "Deploying: $name ($ip, ${vcpu}vCPU, ${mem}GB RAM)" | tee -a "$LOG_FILE"
  
  # Create cloud-init directory
  mkdir -p "$CLOUD_DIR/$name"
  
  # Meta-data
  cat > "$CLOUD_DIR/$name/meta-data" << EOF
instance-id: $name
local-hostname: $name
EOF

  # User-data
  cat > "$CLOUD_DIR/$name/user-data" << EOF
#cloud-config
hostname: $name
fqdn: $name.demo.local

users:
  - name: $user
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    groups: [sudo, docker]
    lock_passwd: false
    plain_text_passwd: '$DEFAULT_PASS'

package_update: true
package_upgrade: true

packages:
  - qemu-guest-agent
  - curl
  - wget
  - vim
  - git
  - jq
  - htop
  - python3
  - python3-pip
  - docker.io

runcmd:
  - systemctl enable --now qemu-guest-agent
  - systemctl enable --now docker
  - usermod -aG docker $user
  - hostnamectl set-hostname $name

final_message: "$name ready at \$TIMESTAMP"
EOF

  # Generate ISO
  cd "$CLOUD_DIR/$name"
  genisoimage -output "/tmp/$name-init.iso" -volid cidata -joliet -rock user-data meta-data 2>/dev/null || return 1
  
  # Copy disk from template
  cp "$DISK_DIR/demo-template.qcow2" "$DISK_DIR/$name.qcow2" || return 1
  
  # Create VM
  sudo virt-install \
    --name "$name" \
    --vcpus "$vcpu" \
    --memory "$((mem * 1024))" \
    --disk path="$DISK_DIR/$name.qcow2,format=qcow2,bus=virtio" \
    --cdrom "/tmp/$name-init.iso" \
    --os-variant ubuntu24.04 \
    --network network=demo_nat,model=virtio \
    --graphics none \
    --console pty,target_type=serial \
    --noautoconsole \
    --wait 0 2>&1 | tee -a "$LOG_FILE" || true
  
  # Verify VM was created
  if sudo virsh dominfo "$name" &>/dev/null; then
    echo "$name: OK" | tee -a "$LOG_FILE"
    return 0
  else
    echo "$name: FAILED" | tee -a "$LOG_FILE"
    return 1
  fi
}

echo "Deploying ${#VM_DEFS[@]} VMs..." | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

DEPLOYED=0
FAILED=0

for vm_def in "${VM_DEFS[@]}"; do
  IFS=':' read -r name ip vcpu mem user desc <<< "$vm_def"
  
  if deploy_vm "$name" "$ip" "$vcpu" "$mem" "$user"; then
    ((DEPLOYED++))
  else
    echo "$name: FAILED" | tee -a "$LOG_FILE"
    ((FAILED++))
  fi
  
  sleep 3
done

echo "" | tee -a "$LOG_FILE"
echo "=== Deployment Summary ===" | tee -a "$LOG_FILE"
echo "Deployed: $DEPLOYED" | tee -a "$LOG_FILE"
echo "Failed: $FAILED" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

echo "Waiting 60 seconds for VMs to boot..." | tee -a "$LOG_FILE"
sleep 60

echo "" | tee -a "$LOG_FILE"
echo "=== VM Status ===" | tee -a "$LOG_FILE"
sudo virsh list --all | grep demo | tee -a "$LOG_FILE"

echo "" | tee -a "$LOG_FILE"
echo "=== DHCP Leases ===" | tee -a "$LOG_FILE"
sudo virsh net-dhcp-leases demo_nat | tee -a "$LOG_FILE"

echo "" | tee -a "$LOG_FILE"
echo "=== Deployment Complete ===" | tee -a "$LOG_FILE"
echo "Password: $DEFAULT_PASS" | tee -a "$LOG_FILE"
echo "Network: 192.168.200.0/24" | tee -a "$LOG_FILE"