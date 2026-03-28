---
name: system-ops
description: Use when writing code that executes iptables/nftables/ss system commands, parses their output, classifies port exposure, or validates firewall rule inputs. Security-critical — read fully before writing any exec code.
---

# System Operations — Security-Critical

## Allowed Command Whitelist

| Command | Allowed Args | Purpose |
|---------|-------------|---------|
| `iptables` | `-S`, `-A`, `-I`, `-D`, `-L -n` | Read/write IPv4 filter |
| `iptables-save` | (none) | Dump all rules |
| `ip6tables` | `-S` | Read IPv6 rules |
| `ss` | `-ltnp`, `-lunp` | List listening ports |
| `nft` | `list ruleset` | Read nftables |
| `ip` | `route`, `addr` | Network info |

**Never use `bash -c`, `sh -c`, or any shell with user-controlled strings.**

## Input Validation Before Any iptables Write

```go
func validateIP(ip string) error {
    if ip == "" { return nil }
    if _, _, err := net.ParseCIDR(ip); err != nil {
        if net.ParseIP(ip) == nil {
            return fmt.Errorf("invalid IP or CIDR: %q", ip)
        }
    }
    return nil
}

func validatePort(port int) error {
    if port < 0 || port > 65535 {
        return fmt.Errorf("port %d out of range [0,65535]", port)
    }
    return nil
}

var allowedActions = map[string]bool{"ACCEPT": true, "DROP": true, "REJECT": true}

// Chain: uppercase letters, digits, hyphens only
var chainRe = regexp.MustCompile(`^[A-Z][A-Z0-9\-]{0,28}$`)
```

## Safe iptables Arg Construction

```go
func buildArgs(r *model.FirewallRule) []string {
    args := []string{"-I", r.Chain}
    if r.SrcIP != ""     { args = append(args, "-s", r.SrcIP) }
    if r.Protocol != "all" { args = append(args, "-p", r.Protocol) }
    if r.DstPort > 0     { args = append(args, "--dport", strconv.Itoa(r.DstPort)) }
    args = append(args, "-j", r.Action)
    if r.Comment != "" {
        // strip shell metacharacters from comment
        safe := regexp.MustCompile(`[^a-zA-Z0-9 _\-\.]`).ReplaceAllString(r.Comment, "")
        args = append(args, "-m", "comment", "--comment", safe)
    }
    return args
}
```

Each element is a discrete arg — no shell interpolation.

## ss Output Parsing

`ss -ltnp` / `ss -lunp` format:
```
State  Recv-Q Send-Q  Local Address:Port  Peer Address:Port  Process
LISTEN 0      4096    0.0.0.0:8080        0.0.0.0:*          users:(("docker-proxy",pid=1148773,fd=7))
LISTEN 0      4096    [::]:8081           [::]:*             users:(("docker-proxy",pid=3631914,fd=7))
```

Parsing rules:
- Skip header line and empty lines
- Local addr+port: split on last `:` — IPv6 addresses wrapped in `[...]`
- Process field regex: `users:\(\("([^"]+)",pid=(\d+)`
- One port may have multiple processes in the Process field

## Exposure Level Classification

```go
func classifyExposure(addr string) string {
    switch addr {
    case "0.0.0.0", "::", "*", "": return "public"
    case "127.0.0.1", "::1":       return "loopback"
    }
    ip := net.ParseIP(addr)
    if ip == nil { return "specific" }
    for _, cidr := range []string{
        "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7",
    } {
        _, net_, _ := net.ParseCIDR(cidr)
        if net_.Contains(ip) { return "private" }
    }
    return "specific"
}
```

## Source Type Classification

```go
var dockerProcs = []string{"docker-proxy", "dockerd", "containerd"}
var systemProcs = []string{"sshd", "nginx", "dnsmasq", "systemd",
                           "chronyd", "rsyslogd", "cron", "avahi"}

func classifySource(name string) string {
    lower := strings.ToLower(name)
    for _, p := range dockerProcs {
        if strings.Contains(lower, p) { return "docker" }
    }
    for _, p := range systemProcs {
        if strings.Contains(lower, p) { return "system" }
    }
    return "user"
}
```

## nftables Graceful Degradation

```go
out, err := execCmd(ctx, "nft", "list", "ruleset")
if err != nil {
    // not installed or no permission — return available:false, not 500
    return &NftResult{Available: false}, nil
}
return &NftResult{Available: true, Raw: out}, nil
```

## Mandatory Logging for Writes

```go
slog.Info("iptables write",
    "operation", "INSERT",
    "chain", rule.Chain,
    "src_ip", rule.SrcIP,
    "dst_port", rule.DstPort,
    "action", rule.Action,
    "rule_id", rule.ID,
)
```

Log errors at ERROR level with full args context.
