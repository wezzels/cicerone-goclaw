package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// securityCmd represents the security command
var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Run security audit",
	Long: `Run security audit on the system.

Audits:
  - SSH configuration (password auth, root login)
  - Firewall status (UFW)
  - Open ports
  - User permissions (sudo/wheel)
  - Sensitive file permissions
  - Package security updates
  - Failed login attempts`,
	RunE: runSecurity,
}

type AuditResult struct {
	Name     string
	Severity string // HIGH, MED, LOW, OK
	Detail   string
	Action   string
}

func init() {
	rootCmd.AddCommand(securityCmd)
}

func runSecurity(cmd *cobra.Command, args []string) error {
	fmt.Println("🔒 Security Audit")
	fmt.Println("=================")
	fmt.Println()

	audits := []struct {
		Name string
		Fn   func() AuditResult
	}{
		{"SSH Config", auditSSH},
		{"Firewall (UFW)", auditFirewall},
		{"Open Ports", auditPorts},
		{"User Permissions", auditUsers},
		{"File Permissions", auditFiles},
		{"Package Updates", auditUpdates},
		{"Failed Logins", auditLogins},
	}

	high := 0
	medium := 0
	low := 0
	ok := 0

	for _, audit := range audits {
		result := audit.Fn()
		printAuditResult(result)

		switch result.Severity {
		case "HIGH":
			high++
		case "MED":
			medium++
		case "LOW":
			low++
		case "OK":
			ok++
		}
	}

	fmt.Println()
	fmt.Println("=================")
	fmt.Printf("Severity: %d HIGH, %d MEDIUM, %d LOW, %d OK\n", high, medium, low, ok)

	if high > 0 {
		fmt.Println("\n⚠️  Action required: Address HIGH severity issues")
		return fmt.Errorf("security issues found")
	}

	return nil
}

func printAuditResult(r AuditResult) {
	var color, icon string
	switch r.Severity {
	case "HIGH":
		color = "\033[31m" // Red
		icon = "!"
	case "MED":
		color = "\033[33m" // Yellow
		icon = "?"
	case "LOW":
		color = "\033[36m" // Cyan
		icon = "·"
	case "OK":
		color = "\033[32m" // Green
		icon = "✓"
	}
	reset := "\033[0m"

	fmt.Printf("  %s[%s]%s %-25s %s\n", color, icon, reset, r.Name+":", r.Detail)
	if r.Action != "" {
		fmt.Printf("         → %s\n", r.Action)
	}
}

func auditSSH() AuditResult {
	// Check SSH config
	data, err := os.ReadFile("/etc/ssh/sshd_config")
	if err != nil {
		return AuditResult{"SSH Config", "LOW", "cannot read config", ""}
	}

	config := string(data)

	// Check for password authentication
	if strings.Contains(config, "PasswordAuthentication yes") {
		return AuditResult{"SSH Config", "HIGH", "password auth enabled", "Set PasswordAuthentication no"}
	}

	// Check for root login
	if strings.Contains(config, "PermitRootLogin yes") {
		return AuditResult{"SSH Config", "HIGH", "root login enabled", "Set PermitRootLogin no"}
	}

	return AuditResult{"SSH Config", "OK", "key-only auth configured", ""}
}

func auditFirewall() AuditResult {
	// Check UFW status
	out, err := exec.Command("ufw", "status").Output()
	if err != nil {
		return AuditResult{"Firewall (UFW)", "MED", "UFW not available", "Install ufw"}
	}

	status := string(out)
	if strings.Contains(status, "Status: active") {
		// Count allowed ports
		lines := strings.Split(status, "\n")
		allowed := 0
		for _, line := range lines {
			if strings.Contains(line, "ALLOW") {
				allowed++
			}
		}
		return AuditResult{"Firewall (UFW)", "OK", fmt.Sprintf("active (%d rules)", allowed), ""}
	}

	if strings.Contains(status, "Status: inactive") {
		return AuditResult{"Firewall (UFW)", "HIGH", "inactive", "Run 'sudo ufw enable'"}
	}

	return AuditResult{"Firewall (UFW)", "MED", "unknown status", ""}
}

func auditPorts() AuditResult {
	// Check listening ports
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return AuditResult{"Open Ports", "LOW", "cannot check", ""}
	}

	lines := strings.Split(string(out), "\n")
	ports := []string{}
	for _, line := range lines {
		if strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				// Extract port from local address
				addr := fields[3]
				if strings.Contains(addr, ":") {
					parts := strings.Split(addr, ":")
					port := parts[len(parts)-1]
					ports = append(ports, port)
				}
			}
		}
	}

	if len(ports) == 0 {
		return AuditResult{"Open Ports", "OK", "no ports listening", ""}
	}

	uniquePorts := unique(ports)
	if len(uniquePorts) > 10 {
		return AuditResult{"Open Ports", "MED", fmt.Sprintf("%d ports open", len(uniquePorts)), "Review with 'ss -tlnp'"}
	}

	return AuditResult{"Open Ports", "OK", fmt.Sprintf("ports: %s", strings.Join(uniquePorts, ", ")), ""}
}

func auditUsers() AuditResult {
	// Check sudo/wheel group members
	out, err := exec.Command("getent", "group", "sudo").Output()
	if err != nil {
		// Try wheel group
		out, err = exec.Command("getent", "group", "wheel").Output()
		if err != nil {
			return AuditResult{"User Permissions", "LOW", "cannot check", ""}
		}
	}

	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, ":")
	if len(parts) >= 4 {
		users := strings.Split(parts[3], ",")
		if len(users) > 3 {
			return AuditResult{"User Permissions", "MED", fmt.Sprintf("%d sudo users", len(users)), "Review sudo membership"}
		}
		return AuditResult{"User Permissions", "OK", fmt.Sprintf("%d sudo users", len(users)), ""}
	}

	return AuditResult{"User Permissions", "OK", "no sudo users", ""}
}

func auditFiles() AuditResult {
	// Check sensitive file permissions
	sensitiveFiles := []string{
		"/etc/shadow",
		"/etc/gshadow",
		"/etc/sudoers",
	}

	issues := []string{}
	for _, file := range sensitiveFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		mode := info.Mode()
		// Check if world-readable
		if mode&0004 != 0 {
			issues = append(issues, file)
		}
	}

	if len(issues) > 0 {
		return AuditResult{"File Permissions", "HIGH", fmt.Sprintf("world-readable: %s", strings.Join(issues, ", ")), "Fix permissions with chmod"}
	}

	return AuditResult{"File Permissions", "OK", "sensitive files protected", ""}
}

func auditUpdates() AuditResult {
	// Check for security updates (Debian/Ubuntu)
	out, err := exec.Command("apt", "list", "--upgradable").Output()
	if err != nil {
		return AuditResult{"Package Updates", "LOW", "cannot check", ""}
	}

	lines := strings.Split(string(out), "\n")
	updates := 0
	for _, line := range lines {
		if strings.Contains(line, "security") {
			updates++
		}
	}

	if updates > 0 {
		return AuditResult{"Package Updates", "MED", fmt.Sprintf("%d security updates", updates), "Run 'apt upgrade'"}
	}

	return AuditResult{"Package Updates", "OK", "no security updates", ""}
}

func auditLogins() AuditResult {
	// Check failed login attempts (lastb)
	out, err := exec.Command("sh", "-c", "lastb -n 10 2>/dev/null | wc -l").Output()
	if err != nil {
		return AuditResult{"Failed Logins", "LOW", "cannot check", ""}
	}

	count := strings.TrimSpace(string(out))
	var num int
	fmt.Sscanf(count, "%d", &num)

	// lastb output includes header line
	if num > 5 {
		return AuditResult{"Failed Logins", "MED", fmt.Sprintf("%d failed attempts", num), "Check 'lastb' for details"}
	}

	return AuditResult{"Failed Logins", "OK", fmt.Sprintf("%d failed attempts", num), ""}
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}