#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RULES_FILE="${RULES_FILE:-$SCRIPT_DIR/rules.conf}"

CHAIN_V4="FIREWALL-INBOUND"
CHAIN_V6="FIREWALL-INBOUND-V6"
DOCKER_CHAIN_V4="FIREWALL-DOCKER"
DOCKER_CHAIN_V6="FIREWALL-DOCKER-V6"

ACTION="${1:---apply}"

IPTABLES_BIN="${IPTABLES_BIN:-iptables}"
IP6TABLES_BIN="${IP6TABLES_BIN:-ip6tables}"

SCRIPT_PATH="${SCRIPT_PATH:-$SCRIPT_DIR/firewall-apply.sh}"
SERVICE_NAME="${SERVICE_NAME:-firewall-apply.service}"
SERVICE_PATH="${SERVICE_PATH:-/etc/systemd/system/$SERVICE_NAME}"

usage() {
  cat <<'EOF'
Usage:
  bash firewall-apply.sh
  bash firewall-apply.sh --apply
  bash firewall-apply.sh --dry-run
  bash firewall-apply.sh --flush
  bash firewall-apply.sh --install-service

Environment overrides:
  RULES_FILE=/path/to/rules.conf
  IPTABLES_BIN=iptables
  IP6TABLES_BIN=ip6tables
  SCRIPT_PATH=/path/to/firewall-apply.sh
  SERVICE_NAME=firewall-apply.service
  SERVICE_PATH=/etc/systemd/system/firewall-apply.service
EOF
}

require_files() {
  if [[ ! -f "$RULES_FILE" ]]; then
    echo "Missing config file: $RULES_FILE" >&2
    exit 1
  fi

  if ! command -v "$IPTABLES_BIN" >/dev/null 2>&1; then
    echo "Missing command: $IPTABLES_BIN" >&2
    exit 1
  fi
}

append_rule() {
  local -n ref=$1
  ref+=("$2")
}

is_ipv6() {
  [[ "$1" == *:* ]]
}

chain_exists_v4() {
  "$IPTABLES_BIN" -S "$1" >/dev/null 2>&1
}

chain_exists_v6() {
  command -v "$IP6TABLES_BIN" >/dev/null 2>&1 && "$IP6TABLES_BIN" -S "$1" >/dev/null 2>&1
}

parse_rules_conf() {
  local section=""
  local line trimmed

  GLOBAL_ENTRIES=()
  PUBLIC_ENTRIES=()
  INBOUND_ENTRIES=()

  while IFS= read -r line; do
    trimmed="${line#"${line%%[![:space:]]*}"}"
    [[ -z "${trimmed// }" ]] && continue
    [[ "$trimmed" =~ ^# ]] && continue

    case "$trimmed" in
      "[global]") section="global"; continue ;;
      "[public]") section="public"; continue ;;
      "[inbound]") section="inbound"; continue ;;
    esac

    case "$section" in
      global) GLOBAL_ENTRIES+=("$trimmed") ;;
      public) PUBLIC_ENTRIES+=("$trimmed") ;;
      inbound) INBOUND_ENTRIES+=("$trimmed") ;;
      *)
        echo "Entry found outside section in $RULES_FILE: $trimmed" >&2
        exit 1
        ;;
    esac
  done < "$RULES_FILE"
}

load_sources_from_entries() {
  local family=$1
  local -n entries=$2
  local -n out=$3
  local source

  for source in "${entries[@]}"; do
    if [[ "$family" == "v4" && $(is_ipv6 "$source" && echo yes || echo no) == "yes" ]]; then
      continue
    fi
    if [[ "$family" == "v6" && $(is_ipv6 "$source" && echo yes || echo no) == "no" ]]; then
      continue
    fi
    out["$source"]=1
  done
}

load_ports_from_entries() {
  local -n entries=$1
  local -n out=$2
  local entry port proto

  for entry in "${entries[@]}"; do
    read -r port proto <<<"$entry"
    if [[ -z "${port:-}" || -z "${proto:-}" ]]; then
      echo "Invalid port entry: $entry" >&2
      exit 1
    fi
    out["$proto:$port"]=1
  done
}

load_inbound_rules() {
  local family=$1
  local -n ports_out=$2
  local -n rules_out=$3
  local entry port proto source

  for entry in "${INBOUND_ENTRIES[@]}"; do
    read -r port proto source <<<"$entry"
    if [[ -z "${port:-}" || -z "${proto:-}" || -z "${source:-}" ]]; then
      echo "Invalid inbound entry: $entry" >&2
      exit 1
    fi
    if [[ "$family" == "v4" && $(is_ipv6 "$source" && echo yes || echo no) == "yes" ]]; then
      continue
    fi
    if [[ "$family" == "v6" && $(is_ipv6 "$source" && echo yes || echo no) == "no" ]]; then
      continue
    fi
    ports_out["$proto:$port"]=1
    rules_out+=("$proto:$port:$source")
  done
}

has_v6_need() {
  local item

  if ! command -v "$IP6TABLES_BIN" >/dev/null 2>&1; then
    return 1
  fi

  for item in "${GLOBAL_ENTRIES[@]}" "${INBOUND_ENTRIES[@]}"; do
    if [[ "$item" == *:* ]]; then
      return 0
    fi
  done

  if [[ ${#PUBLIC_ENTRIES[@]} -gt 0 ]]; then
    return 0
  fi

  return 1
}

should_manage_docker_v4() {
  chain_exists_v4 "DOCKER-USER"
}

should_manage_docker_v6() {
  chain_exists_v6 "DOCKER-USER"
}

build_chain_v4() {
  local chain=$1
  local attach_parent=$2
  local docker_mode=$3
  local -n out=$4

  local -A global_sources=()
  local -A public_ports=()
  local -A protected_ports=()
  local inbound_rules=()
  local key proto port source entry

  if [[ "$docker_mode" == "docker" ]] && ! should_manage_docker_v4; then
    return
  fi

  append_rule out "$IPTABLES_BIN -N $chain 2>/dev/null || true"
  append_rule out "$IPTABLES_BIN -F $chain"
  append_rule out "$IPTABLES_BIN -C $attach_parent -j $chain 2>/dev/null || $IPTABLES_BIN -I $attach_parent 1 -j $chain"
  append_rule out "$IPTABLES_BIN -A $chain -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT"

  if [[ "$docker_mode" == "docker" ]]; then
    append_rule out "$IPTABLES_BIN -A $chain -i docker0 -j RETURN"
    append_rule out "$IPTABLES_BIN -A $chain -i br+ -j RETURN"
  else
    append_rule out "$IPTABLES_BIN -A $chain -i lo -j ACCEPT"
  fi

  load_sources_from_entries v4 GLOBAL_ENTRIES global_sources
  for source in "${!global_sources[@]}"; do
    append_rule out "$IPTABLES_BIN -A $chain -s $source -j ACCEPT"
  done

  load_ports_from_entries PUBLIC_ENTRIES public_ports
  for key in "${!public_ports[@]}"; do
    proto="${key%%:*}"
    port="${key##*:}"
    append_rule out "$IPTABLES_BIN -A $chain -p $proto --dport $port -j ACCEPT"
    protected_ports["$key"]=1
  done

  load_inbound_rules v4 protected_ports inbound_rules
  for entry in "${inbound_rules[@]}"; do
    proto="${entry%%:*}"
    entry="${entry#*:}"
    port="${entry%%:*}"
    source="${entry#*:}"
    append_rule out "$IPTABLES_BIN -A $chain -p $proto --dport $port -s $source -j ACCEPT"
  done

  for key in "${!protected_ports[@]}"; do
    proto="${key%%:*}"
    port="${key##*:}"
    append_rule out "$IPTABLES_BIN -A $chain -p $proto --dport $port -j DROP"
  done

  append_rule out "$IPTABLES_BIN -A $chain -p tcp -j DROP"
  append_rule out "$IPTABLES_BIN -A $chain -p udp -j DROP"
  append_rule out "$IPTABLES_BIN -A $chain -j RETURN"
}

build_chain_v6() {
  local chain=$1
  local attach_parent=$2
  local docker_mode=$3
  local -n out=$4

  local -A global_sources=()
  local -A public_ports=()
  local -A protected_ports=()
  local inbound_rules=()
  local key proto port source entry

  if ! has_v6_need; then
    return
  fi

  if [[ "$docker_mode" == "docker" ]] && ! should_manage_docker_v6; then
    return
  fi

  append_rule out "$IP6TABLES_BIN -N $chain 2>/dev/null || true"
  append_rule out "$IP6TABLES_BIN -F $chain"
  append_rule out "$IP6TABLES_BIN -C $attach_parent -j $chain 2>/dev/null || $IP6TABLES_BIN -I $attach_parent 1 -j $chain"
  append_rule out "$IP6TABLES_BIN -A $chain -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT"

  if [[ "$docker_mode" == "docker" ]]; then
    append_rule out "$IP6TABLES_BIN -A $chain -i docker0 -j RETURN"
    append_rule out "$IP6TABLES_BIN -A $chain -i br+ -j RETURN"
  else
    append_rule out "$IP6TABLES_BIN -A $chain -i lo -j ACCEPT"
  fi

  load_sources_from_entries v6 GLOBAL_ENTRIES global_sources
  for source in "${!global_sources[@]}"; do
    append_rule out "$IP6TABLES_BIN -A $chain -s $source -j ACCEPT"
  done

  load_ports_from_entries PUBLIC_ENTRIES public_ports
  for key in "${!public_ports[@]}"; do
    proto="${key%%:*}"
    port="${key##*:}"
    append_rule out "$IP6TABLES_BIN -A $chain -p $proto --dport $port -j ACCEPT"
    protected_ports["$key"]=1
  done

  load_inbound_rules v6 protected_ports inbound_rules
  for entry in "${inbound_rules[@]}"; do
    proto="${entry%%:*}"
    entry="${entry#*:}"
    port="${entry%%:*}"
    source="${entry#*:}"
    append_rule out "$IP6TABLES_BIN -A $chain -p $proto --dport $port -s $source -j ACCEPT"
  done

  for key in "${!protected_ports[@]}"; do
    proto="${key%%:*}"
    port="${key##*:}"
    append_rule out "$IP6TABLES_BIN -A $chain -p $proto --dport $port -j DROP"
  done

  append_rule out "$IP6TABLES_BIN -A $chain -p tcp -j DROP"
  append_rule out "$IP6TABLES_BIN -A $chain -p udp -j DROP"
  append_rule out "$IP6TABLES_BIN -A $chain -j RETURN"
}

flush_rules() {
  cat <<EOF
$IPTABLES_BIN -D INPUT -j $CHAIN_V4 2>/dev/null || true
$IPTABLES_BIN -F $CHAIN_V4 2>/dev/null || true
$IPTABLES_BIN -X $CHAIN_V4 2>/dev/null || true
$IPTABLES_BIN -D DOCKER-USER -j $DOCKER_CHAIN_V4 2>/dev/null || true
$IPTABLES_BIN -F $DOCKER_CHAIN_V4 2>/dev/null || true
$IPTABLES_BIN -X $DOCKER_CHAIN_V4 2>/dev/null || true
$IP6TABLES_BIN -D INPUT -j $CHAIN_V6 2>/dev/null || true
$IP6TABLES_BIN -F $CHAIN_V6 2>/dev/null || true
$IP6TABLES_BIN -X $CHAIN_V6 2>/dev/null || true
$IP6TABLES_BIN -D DOCKER-USER -j $DOCKER_CHAIN_V6 2>/dev/null || true
$IP6TABLES_BIN -F $DOCKER_CHAIN_V6 2>/dev/null || true
$IP6TABLES_BIN -X $DOCKER_CHAIN_V6 2>/dev/null || true
EOF
}

has_systemd() {
  command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]
}

install_service() {
  if ! has_systemd; then
    echo "systemd not available, skip service installation"
    return 0
  fi

  cat > "$SERVICE_PATH" <<EOF
[Unit]
Description=Apply custom firewall rules from $RULES_FILE
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=$SCRIPT_PATH --apply
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" >/dev/null 2>&1 || true
}

main() {
  local cmds=()

  case "$ACTION" in
    --dry-run)
      require_files
      parse_rules_conf
      build_chain_v4 "$CHAIN_V4" "INPUT" "host" cmds
      build_chain_v4 "$DOCKER_CHAIN_V4" "DOCKER-USER" "docker" cmds
      build_chain_v6 "$CHAIN_V6" "INPUT" "host" cmds
      build_chain_v6 "$DOCKER_CHAIN_V6" "DOCKER-USER" "docker" cmds
      printf '%s\n' "${cmds[@]}"
      ;;
    --apply)
      require_files
      parse_rules_conf
      build_chain_v4 "$CHAIN_V4" "INPUT" "host" cmds
      build_chain_v4 "$DOCKER_CHAIN_V4" "DOCKER-USER" "docker" cmds
      build_chain_v6 "$CHAIN_V6" "INPUT" "host" cmds
      build_chain_v6 "$DOCKER_CHAIN_V6" "DOCKER-USER" "docker" cmds
      for cmd in "${cmds[@]}"; do
        echo "+ $cmd"
        eval "$cmd"
      done
      install_service
      ;;
    --install-service)
      require_files
      install_service
      ;;
    --flush)
      flush_rules
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main
