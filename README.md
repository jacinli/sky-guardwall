# Sky Guardwall

Sky Guardwall is a VPS firewall control project focused on one practical goal:

- manage public ingress rules with a simple config file
- avoid breaking Docker internal traffic
- avoid interfering with normal container-to-container communication
- stay usable on common Debian and Ubuntu hosts

This repository also includes a lightweight firewall bootstrap script that can be reused on standalone servers.

## Included Firewall Files

Public-safe firewall files are included here:

- `scripts/firewall/firewall-apply.sh`
- `scripts/firewall/rules.conf.example`

The example file contains placeholders only and does not include any real IP addresses.

## Rule Model

The firewall config uses three sections:

```txt
[global]
[public]
[inbound]
```

Meaning:

- `global`
  - sources that are allowed to access all ports
- `public`
  - ports open to everyone
- `inbound`
  - port-level allowlist rules

Example:

```txt
[global]
YOUR_IPV4_A
YOUR_IPV4_B
YOUR_IPV6_CIDR_A

[public]
80 tcp
443 tcp

[inbound]
9176 tcp YOUR_IPV4_A
3000 tcp YOUR_IPV4_A
8080 tcp YOUR_IPV4_A
```

## Design Notes

This script is intentionally conservative:

- it manages host ingress and Docker published ports
- it skips Docker bridge traffic
- it does not try to control internal container networking
- it skips Docker-specific handling when `DOCKER-USER` does not exist
- it skips IPv6 when `ip6tables` is not available

That keeps it practical for real VPS use, especially on mixed Debian and Ubuntu environments.

## Usage

Preview:

```bash
bash scripts/firewall/firewall-apply.sh --dry-run
```

Apply:

```bash
bash scripts/firewall/firewall-apply.sh
```

Flush:

```bash
bash scripts/firewall/firewall-apply.sh --flush
```

The script can also install a systemd unit automatically so rules are restored after reboot.
