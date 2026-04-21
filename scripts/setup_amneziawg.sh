#!/bin/bash
# setup_amneziawg.sh
#
# Deploys AmneziaWG on a fresh Ubuntu 22.04 server with the same configuration
# as the existing CH-1 node (45.12.111.254):
#   - Port 443/UDP
#   - Obfuscation: Jc=5 Jmin=40 Jmax=75 S1=28 S2=67 H1-H4 matching client builds
#   - 10.8.0.0/24 VPN subnet, 100 mbit/s per-client rate limiting
#   - wg-manage.sh + wg-ratelimit.sh lifecycle scripts
#   - komkov.vv SSH user authorised from main server (185.104.114.242)
#
# Usage:
#   scp scripts/setup_amneziawg.sh root@<new-server>:/root/
#   ssh root@<new-server> bash /root/setup_amneziawg.sh
#
# After setup, add the server to the DB (type = 'amneziawg') and .env is
# already configured (AWG_SSH_USER=komkov.vv, paths use defaults).

set -euo pipefail

# ──────────────────────────────────────────────
#  CONSTANTS  (match existing CH-1 node exactly)
# ──────────────────────────────────────────────
PORT=443
WG_IFACE="wg0"
VPN_ADDR="10.8.0.1/24"
DNS="1.1.1.1"
MTU=1280

# AmneziaWG obfuscation params — MUST match what clients ship
JC=5; JMIN=40; JMAX=75; S1=28; S2=67
H1=1683873075; H2=601371293; H3=1062143158; H4=2977900877

# SSH user the Go project connects as (= OVPN_SSH_USER / AWG_SSH_USER in .env)
SSH_USER="komkov.vv"

AWG_CONF="/etc/amnezia/amneziawg/wg0.conf"
WG_CONF_MIRROR="/etc/wireguard/wg0.conf"   # kept in sync by wg-manage.sh
CLIENT_DIR="/etc/wireguard/clients"
MANAGE="/usr/local/bin/wg-manage.sh"
RATELIMIT="/usr/local/bin/wg-ratelimit.sh"

# ──────────────────────────────────────────────
#  HELPERS
# ──────────────────────────────────────────────
log()  { echo -e "\033[1;32m[+]\033[0m $*"; }
warn() { echo -e "\033[1;33m[!]\033[0m $*"; }
die()  { echo -e "\033[1;31m[✗]\033[0m $*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] && die "Run as root (sudo bash $0)"
[ -f /etc/os-release ] && source /etc/os-release
[[ "${ID:-}" == "ubuntu" && "${VERSION_ID:-}" == "22.04" ]] \
    || warn "Tested on Ubuntu 22.04 only; continuing anyway"

# Auto-detect WAN interface and public IP
WAN_IFACE=$(ip route show default 2>/dev/null | awk '/default/ {print $5; exit}')
[ -z "$WAN_IFACE" ] && die "Cannot detect default WAN interface"
log "WAN interface: $WAN_IFACE"

SERVER_IP=$(curl -s --connect-timeout 10 https://api.ipify.org \
         || curl -s --connect-timeout 10 https://ifconfig.me \
         || ip route get 8.8.8.8 2>/dev/null | awk '{print $7; exit}')
[ -z "$SERVER_IP" ] && die "Cannot detect public IP"
log "Public IP: $SERVER_IP"

# ──────────────────────────────────────────────
#  SYSTEM DEPS
# ──────────────────────────────────────────────
log "Installing build dependencies..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
    linux-headers-$(uname -r) \
    build-essential git make pkg-config \
    libelf-dev bc dkms curl iptables python3

# ──────────────────────────────────────────────
#  AMNEZIAWG KERNEL MODULE
# ──────────────────────────────────────────────
if lsmod | grep -q amneziawg; then
    log "AmneziaWG kernel module already loaded — skipping build"
else
    log "Building AmneziaWG kernel module (this takes ~3 min)..."
    TMPMOD=$(mktemp -d)
    git clone --depth 1 \
        https://github.com/amnezia-vpn/amneziawg-linux-kernel-module.git \
        "$TMPMOD/awg-module"
    cd "$TMPMOD/awg-module"
    make -j"$(nproc)"
    make install
    modprobe amneziawg
    cd /
    rm -rf "$TMPMOD"
    log "Kernel module built and loaded"
fi

echo "amneziawg" > /etc/modules-load.d/amneziawg.conf

# ──────────────────────────────────────────────
#  AWG TOOLS (awg binary + awg-quick script)
# ──────────────────────────────────────────────
if ! command -v awg &>/dev/null; then
    log "Building awg tools..."
    TMPTOOLS=$(mktemp -d)
    git clone --depth 1 \
        https://github.com/amnezia-vpn/amneziawg-tools.git \
        "$TMPTOOLS/awg-tools"
    cd "$TMPTOOLS/awg-tools/src"
    make -j"$(nproc)"
    install -m 755 awg /usr/local/bin/awg
    # awg-quick is the bash script from wg-quick/linux.bash
    install -m 755 ../contrib/external-tests/wg-quick/linux.bash \
        /usr/local/bin/awg-quick 2>/dev/null \
    || cp wg-quick/linux.bash /usr/local/bin/awg-quick 2>/dev/null \
    || true
    cd /
    rm -rf "$TMPTOOLS"
    log "awg tools installed"
else
    log "awg already installed: $(awg --version 2>/dev/null || echo 'ok')"
fi

# If awg-quick wasn't found in the tools tree, fetch the known-good version
if ! command -v awg-quick &>/dev/null; then
    log "Fetching awg-quick script..."
    curl -fsSL \
        "https://raw.githubusercontent.com/amnezia-vpn/amneziawg-tools/master/src/wg-quick/linux.bash" \
        -o /usr/local/bin/awg-quick
    chmod 755 /usr/local/bin/awg-quick
fi

# ──────────────────────────────────────────────
#  SERVER KEY PAIR
# ──────────────────────────────────────────────
log "Generating server key pair..."
SERVER_PRIV=$(awg genkey)
SERVER_PUB=$(echo "$SERVER_PRIV" | awg pubkey)
log "Server public key: $SERVER_PUB"

# ──────────────────────────────────────────────
#  WG0 CONFIG
# ──────────────────────────────────────────────
log "Writing wg0.conf..."
mkdir -p "$(dirname "$AWG_CONF")" "$CLIENT_DIR" /etc/wireguard

cat > "$AWG_CONF" <<EOF
[Interface]
Address = $VPN_ADDR
ListenPort = $PORT
PrivateKey = $SERVER_PRIV
Jc = $JC
Jmin = $JMIN
Jmax = $JMAX
S1 = $S1
S2 = $S2
H1 = $H1
H2 = $H2
H3 = $H3
H4 = $H4
MTU = $MTU
PostUp = sysctl -w net.ipv4.ip_forward=1
PostUp = iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
PostUp = iptables -A FORWARD -i $WG_IFACE -j ACCEPT
PostUp = iptables -t nat -A POSTROUTING -o $WAN_IFACE -j MASQUERADE
PostUp = iptables -A INPUT -p udp --dport $PORT -j ACCEPT
PostDown = sysctl -w net.ipv4.ip_forward=0
PostDown = iptables -D FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null
PostDown = iptables -D FORWARD -i $WG_IFACE -j ACCEPT 2>/dev/null
PostDown = iptables -t nat -D POSTROUTING -o $WAN_IFACE -j MASQUERADE 2>/dev/null
PostDown = iptables -D INPUT -p udp --dport $PORT -j ACCEPT 2>/dev/null
EOF

chmod 600 "$AWG_CONF"
cp "$AWG_CONF" "$WG_CONF_MIRROR"

# ──────────────────────────────────────────────
#  WG-MANAGE.SH
# ──────────────────────────────────────────────
log "Installing wg-manage.sh..."

# NOTE: this heredoc is unquoted so $SERVER_IP/$PORT expand now (server-specific).
# All other $ inside wg-manage.sh use single-quoted heredocs or escaped vars.
cat > "$MANAGE" << OUTER
#!/bin/bash
# AmneziaWG — client lifecycle management
# Usage: sudo wg-manage.sh [add|remove|list|show|qr] <name>

AWG_IFACE="wg0"
SERVER_IP="$SERVER_IP"
SERVER_PORT="$PORT"
CLIENT_DIR="$CLIENT_DIR"
AWG_CONF="$AWG_CONF"
DNS="$DNS"

JC=$JC; JMIN=$JMIN; JMAX=$JMAX; S1=$S1; S2=$S2
H1=$H1; H2=$H2; H3=$H3; H4=$H4

server_pub() { awg show "\$AWG_IFACE" public-key 2>/dev/null; }

next_ip() {
    for i in \$(seq 2 254); do
        ip="10.8.0.\$i"
        if ! awg show "\$AWG_IFACE" allowed-ips 2>/dev/null | grep -q "\$ip"; then
            echo "\$ip"; return
        fi
    done
    echo "No free IPs" >&2; exit 1
}

cmd_add() {
    local name="\$1"
    [ -z "\$name" ] && { echo "Usage: \$0 add <name>"; exit 1; }
    [ -f "\$CLIENT_DIR/\$name.conf" ] && { echo "Client '\$name' already exists"; exit 1; }

    local priv pub ip
    priv=\$(awg genkey)
    pub=\$(echo "\$priv" | awg pubkey)
    ip=\$(next_ip)

    awg set "\$AWG_IFACE" peer "\$pub" allowed-ips "\$ip/32"

    cat >> "\$AWG_CONF" << EOF

[Peer]
# \$name
PublicKey = \$pub
AllowedIPs = \$ip/32
EOF
    cp "\$AWG_CONF" /etc/wireguard/wg0.conf

    cat > "\$CLIENT_DIR/\$name.conf" << EOF
[Interface]
PrivateKey = \$priv
Address = \$ip/24
DNS = \$DNS
MTU = 1280
Jc = \$JC
Jmin = \$JMIN
Jmax = \$JMAX
S1 = \$S1
S2 = \$S2
H1 = \$H1
H2 = \$H2
H3 = \$H3
H4 = \$H4

[Peer]
PublicKey = \$(server_pub)
Endpoint = \$SERVER_IP:\$SERVER_PORT
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
EOF
    chmod 600 "\$CLIENT_DIR/\$name.conf"
    echo "=== Client '\$name' created ==="
    echo "IP: \$ip"
    echo ""
    cat "\$CLIENT_DIR/\$name.conf"
    /usr/local/bin/wg-ratelimit.sh apply >&2
}

cmd_remove() {
    local name="\$1"
    [ -z "\$name" ] && { echo "Usage: \$0 remove <name>"; exit 1; }
    local cfg="\$CLIENT_DIR/\$name.conf"
    [ ! -f "\$cfg" ] && { echo "Client '\$name' not found"; exit 1; }

    local pub
    pub=\$(grep '^PrivateKey' "\$cfg" | awk '{print \$3}' | awg pubkey)
    awg set "\$AWG_IFACE" peer "\$pub" remove

    python3 - "\$AWG_CONF" "\$pub" << 'PYEOF'
import sys, re
conf, pub = sys.argv[1], sys.argv[2]
with open(conf) as f:
    text = f.read()
pattern = r'\[Peer\][^\[]*PublicKey\s*=\s*' + re.escape(pub) + r'[^\[]*'
text = re.sub(pattern, '', text, flags=re.DOTALL)
with open(conf, 'w') as f:
    f.write(text)
PYEOF
    cp "\$AWG_CONF" /etc/wireguard/wg0.conf
    rm -f "\$cfg"
    echo "Client '\$name' removed"
    /usr/local/bin/wg-ratelimit.sh apply >&2
}

cmd_list() {
    echo "=== Clients ==="
    local found=0
    for cfg in "\$CLIENT_DIR"/*.conf; do
        [ -f "\$cfg" ] || continue
        found=1
        local name ip pub last ago status
        name=\$(basename "\$cfg" .conf)
        ip=\$(grep '^Address' "\$cfg" | awk '{print \$3}')
        pub=\$(grep '^PrivateKey' "\$cfg" | awk '{print \$3}' | awg pubkey)
        last=\$(awg show "\$AWG_IFACE" latest-handshakes 2>/dev/null | grep "\$pub" | awk '{print \$2}')
        if [ -n "\$last" ] && [ "\$last" != "0" ]; then
            ago=\$(( \$(date +%s) - last ))
            status="\${ago}s ago"
        else
            status="not connected"
        fi
        printf "  %-20s %-15s %s\n" "\$name" "\$ip" "\$status"
    done
    [ \$found -eq 0 ] && echo "  (no clients)"
    echo ""
    echo "=== awg show ==="
    awg show "\$AWG_IFACE"
}

cmd_show() {
    local name="\$1"
    [ -z "\$name" ] && { echo "Usage: \$0 show <name>"; exit 1; }
    local cfg="\$CLIENT_DIR/\$name.conf"
    [ ! -f "\$cfg" ] && { echo "Client '\$name' not found"; exit 1; }
    cat "\$cfg"
}

cmd_qr() {
    local name="\$1"
    [ -z "\$name" ] && { echo "Usage: \$0 qr <name>"; exit 1; }
    local cfg="\$CLIENT_DIR/\$name.conf"
    [ ! -f "\$cfg" ] && { echo "Client '\$name' not found"; exit 1; }
    command -v qrencode &>/dev/null || apt-get install -y qrencode -qq
    qrencode -t ansiutf8 < "\$cfg"
}

case "\$1" in
    add)    cmd_add "\$2" ;;
    remove) cmd_remove "\$2" ;;
    list)   cmd_list ;;
    show)   cmd_show "\$2" ;;
    qr)     cmd_qr "\$2" ;;
    *) echo "Usage: sudo \$0 [add|remove|list|show|qr] <name>" ;;
esac
OUTER

chmod 755 "$MANAGE"

# ──────────────────────────────────────────────
#  WG-RATELIMIT.SH
# ──────────────────────────────────────────────
log "Installing wg-ratelimit.sh..."
cat > "$RATELIMIT" << 'EOF'
#!/bin/bash
# Per-client rate limiting for AmneziaWG via HTB + ifb
IFACE="wg0"
IFB="ifb0"
RATE="100mbit"

apply_all() {
    modprobe ifb numifbs=1 2>/dev/null

    tc qdisc del dev $IFACE root 2>/dev/null
    tc qdisc add dev $IFACE root handle 1: htb default 999
    tc class add dev $IFACE parent 1:0 classid 1:999 htb rate 10gbit

    tc qdisc del dev $IFACE ingress 2>/dev/null
    ip link set $IFB up 2>/dev/null
    tc qdisc del dev $IFB root 2>/dev/null
    tc qdisc add dev $IFB root handle 1: htb default 999
    tc class add dev $IFB parent 1:0 classid 1:999 htb rate 10gbit

    tc qdisc add dev $IFACE handle ffff: ingress
    tc filter add dev $IFACE parent ffff: protocol ip u32 match u32 0 0 \
        action mirred egress redirect dev $IFB

    awg show $IFACE allowed-ips | awk '{print $2}' | cut -d'/' -f1 | while read ip; do
        [ -n "$ip" ] && _add_peer "$ip"
    done
}

_add_peer() {
    local ip="$1"
    local octet
    octet=$(echo "$ip" | awk -F. '{print $4}')
    tc class add dev $IFACE parent 1:0 classid "1:$octet" htb rate $RATE ceil $RATE 2>/dev/null
    tc filter add dev $IFACE parent 1:0 protocol ip prio 1 u32 \
        match ip dst "$ip/32" flowid "1:$octet"
    tc class add dev $IFB parent 1:0 classid "1:$octet" htb rate $RATE ceil $RATE 2>/dev/null
    tc filter add dev $IFB parent 1:0 protocol ip prio 1 u32 \
        match ip src "$ip/32" flowid "1:$octet"
}

case "$1" in
    apply) apply_all ;;
    *) echo "Usage: $0 apply" ;;
esac
EOF
chmod 755 "$RATELIMIT"

# ──────────────────────────────────────────────
#  SYSTEMD SERVICES
# ──────────────────────────────────────────────
log "Creating systemd units..."

cat > /etc/systemd/system/awg-quick.target << 'EOF'
[Unit]
Description=All AmneziaWG interfaces
DefaultDependencies=no
After=network.target
EOF

cat > /etc/systemd/system/awg-quick@.service << 'EOF'
[Unit]
Description=AmneziaWG via awg-quick(8) for %i
After=network-online.target nss-lookup.target
Wants=network-online.target nss-lookup.target
PartOf=awg-quick.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/awg-quick up %i
ExecStop=/usr/local/bin/awg-quick down %i
Environment=WG_ENDPOINT_RESOLUTION_RETRIES=infinity

[Install]
WantedBy=multi-user.target awg-quick.target
EOF

# Concrete instance unit for wg0 (mirrors what is on CH-1)
cat > /etc/systemd/system/awg-quick@wg0.service << 'EOF'
[Unit]
Description=AmneziaWG via awg-quick(8) for wg0
After=network-online.target nss-lookup.target
Wants=network-online.target nss-lookup.target
PartOf=awg-quick.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/awg-quick up wg0
ExecStop=/usr/local/bin/awg-quick down wg0
Environment=WG_ENDPOINT_RESOLUTION_RETRIES=infinity

[Install]
WantedBy=multi-user.target awg-quick.target
EOF

cat > /etc/systemd/system/wg-ratelimit.service << 'EOF'
[Unit]
Description=WireGuard per-client rate limiting
After=network.target sys-subsystem-net-devices-wg0.device
Wants=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/wg-ratelimit.sh apply
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

mkdir -p /etc/systemd/system/awg-quick.target.wants
ln -sf /etc/systemd/system/awg-quick@wg0.service \
    /etc/systemd/system/awg-quick.target.wants/awg-quick@wg0.service

systemctl daemon-reload
systemctl enable awg-quick@wg0
systemctl enable wg-ratelimit

# ──────────────────────────────────────────────
#  SUDOERS
# ──────────────────────────────────────────────
log "Configuring passwordless sudo for $SSH_USER..."
cat > /etc/sudoers.d/awg-vpn << EOF
# MiracleVPN project — allows the API server to manage AmneziaWG over SSH
$SSH_USER ALL=(ALL) NOPASSWD: /usr/local/bin/awg *
$SSH_USER ALL=(ALL) NOPASSWD: /usr/local/bin/wg-manage.sh *
$SSH_USER ALL=(ALL) NOPASSWD: /bin/ls $CLIENT_DIR
$SSH_USER ALL=(ALL) NOPASSWD: /usr/bin/ls $CLIENT_DIR
$SSH_USER ALL=(ALL) NOPASSWD: /bin/cat $AWG_CONF
$SSH_USER ALL=(ALL) NOPASSWD: /usr/bin/cat $AWG_CONF
EOF
chmod 440 /etc/sudoers.d/awg-vpn
visudo -c -f /etc/sudoers.d/awg-vpn || { warn "sudoers syntax error — check /etc/sudoers.d/awg-vpn"; }

# ──────────────────────────────────────────────
#  BRING UP THE INTERFACE
# ──────────────────────────────────────────────
log "Starting AmneziaWG..."
# awg-quick needs to know where to find awg; it lives alongside in /usr/local/bin
export PATH="/usr/local/bin:$PATH"
systemctl start awg-quick@wg0
systemctl start wg-ratelimit

# ──────────────────────────────────────────────
#  FINAL REPORT
# ──────────────────────────────────────────────
echo ""
log "══════════════════════════════════════════════"
log " Setup complete!"
log "══════════════════════════════════════════════"
log " Public IP  : $SERVER_IP"
log " VPN port   : $PORT/UDP"
log " Subnet     : 10.8.0.0/24"
log " SSH user   : $SSH_USER"
log ""
log " Verify the interface is up:"
log "   awg show wg0"
log ""
log " To add this server to the project DB:"
log "   INSERT INTO servers (name, host, region, region_name, type, active)"
log "   VALUES ('<name>', '$SERVER_IP', '<region-code>', '<Region>', 'amneziawg', true);"
log ""
log " Test from the main server (185.104.114.242):"
log "   ssh $SSH_USER@$SERVER_IP 'sudo /usr/local/bin/awg show wg0 dump'"
log "══════════════════════════════════════════════"
