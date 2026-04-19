#!/bin/bash
# AmneziaWG — управление клиентами
# Использование: sudo wg-manage.sh [add|remove|list|show|qr] <имя>

AWG_IFACE="wg0"
SERVER_IP="45.12.111.254"
SERVER_PORT="443"
CLIENT_DIR="/etc/wireguard/clients"
AWG_CONF="/etc/amnezia/amneziawg/wg0.conf"
DNS="1.1.1.1"

# AmneziaWG obfuscation params (должны совпадать с сервером)
JC=5; JMIN=40; JMAX=75; S1=28; S2=67
H1=1683873075; H2=601371293; H3=1062143158; H4=2977900877

server_pub() {
    awg show "$AWG_IFACE" public-key 2>/dev/null
}

next_ip() {
    for i in $(seq 2 254); do
        ip="10.8.0.$i"
        if ! awg show "$AWG_IFACE" allowed-ips 2>/dev/null | grep -q "$ip"; then
            echo "$ip"; return
        fi
    done
    echo "Нет свободных IP" >&2; exit 1
}

cmd_add() {
    local name="$1"
    [ -z "$name" ] && { echo "Использование: $0 add <имя>"; exit 1; }
    [ -f "$CLIENT_DIR/$name.conf" ] && { echo "Клиент '$name' уже существует"; exit 1; }

    local priv pub ip
    priv=$(awg genkey)
    pub=$(echo "$priv" | awg pubkey)
    ip=$(next_ip)

    # Добавить пир в живой интерфейс
    awg set "$AWG_IFACE" peer "$pub" allowed-ips "$ip/32"

    # Добавить пир в конфиг-файл (чтобы пережил перезагрузку)
    cat >> "$AWG_CONF" << EOF

[Peer]
# $name
PublicKey = $pub
AllowedIPs = $ip/32
EOF
    cp "$AWG_CONF" /etc/wireguard/wg0.conf

    # Сохранить клиентский конфиг
    cat > "$CLIENT_DIR/$name.conf" << EOF
[Interface]
PrivateKey = $priv
Address = $ip/24
DNS = $DNS
MTU = 1280
Jc = $JC
Jmin = $JMIN
Jmax = $JMAX
S1 = $S1
S2 = $S2
H1 = $H1
H2 = $H2
H3 = $H3
H4 = $H4

[Peer]
PublicKey = $(server_pub)
Endpoint = $SERVER_IP:$SERVER_PORT
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
EOF
    chmod 600 "$CLIENT_DIR/$name.conf"

    echo "=== Клиент '$name' создан ==="
    echo "IP: $ip"
    echo ""
    cat "$CLIENT_DIR/$name.conf"
}

cmd_remove() {
    local name="$1"
    [ -z "$name" ] && { echo "Использование: $0 remove <имя>"; exit 1; }
    local cfg="$CLIENT_DIR/$name.conf"
    [ ! -f "$cfg" ] && { echo "Клиент '$name' не найден"; exit 1; }

    local pub
    pub=$(grep '^PrivateKey' "$cfg" | awk '{print $3}' | awg pubkey)

    # Удалить из живого интерфейса
    awg set "$AWG_IFACE" peer "$pub" remove

    # Удалить из конфиг-файла (блок [Peer] с этим PublicKey)
    python3 - "$AWG_CONF" "$pub" << 'PYEOF'
import sys, re
conf, pub = sys.argv[1], sys.argv[2]
with open(conf) as f:
    text = f.read()
# Удаляем блок [Peer] содержащий нужный PublicKey
pattern = r'\[Peer\][^\[]*PublicKey\s*=\s*' + re.escape(pub) + r'[^\[]*'
text = re.sub(pattern, '', text, flags=re.DOTALL)
with open(conf, 'w') as f:
    f.write(text)
PYEOF
    cp "$AWG_CONF" /etc/wireguard/wg0.conf
    rm -f "$cfg"
    echo "Клиент '$name' удалён"
}

cmd_list() {
    echo "=== Клиенты ==="
    local found=0
    for cfg in "$CLIENT_DIR"/*.conf; do
        [ -f "$cfg" ] || continue
        found=1
        local name ip last
        name=$(basename "$cfg" .conf)
        ip=$(grep '^Address' "$cfg" | awk '{print $3}')
        pub=$(grep '^PrivateKey' "$cfg" | awk '{print $3}' | awg pubkey)
        last=$(awg show "$AWG_IFACE" latest-handshakes 2>/dev/null | grep "$pub" | awk '{print $2}')
        if [ -n "$last" ] && [ "$last" != "0" ]; then
            ago=$(( $(date +%s) - last ))
            status="${ago}s назад"
        else
            status="не подключён"
        fi
        printf "  %-20s %-15s %s\n" "$name" "$ip" "$status"
    done
    [ $found -eq 0 ] && echo "  (нет клиентов)"
    echo ""
    echo "=== awg show ==="
    awg show "$AWG_IFACE"
}

cmd_show() {
    local name="$1"
    [ -z "$name" ] && { echo "Использование: $0 show <имя>"; exit 1; }
    local cfg="$CLIENT_DIR/$name.conf"
    [ ! -f "$cfg" ] && { echo "Клиент '$name' не найден"; exit 1; }
    cat "$cfg"
}

cmd_qr() {
    local name="$1"
    [ -z "$name" ] && { echo "Использование: $0 qr <имя>"; exit 1; }
    local cfg="$CLIENT_DIR/$name.conf"
    [ ! -f "$cfg" ] && { echo "Клиент '$name' не найден"; exit 1; }
    which qrencode &>/dev/null || apt-get install -y qrencode -qq
    qrencode -t ansiutf8 < "$cfg"
    echo ""
    echo "Конфиг: $cfg"
}

case "$1" in
    add)    cmd_add "$2" ;;
    remove) cmd_remove "$2" ;;
    list)   cmd_list ;;
    show)   cmd_show "$2" ;;
    qr)     cmd_qr "$2" ;;
    *)
        echo "AmneziaWG — управление клиентами"
        echo ""
        echo "Использование: sudo $0 <команда> [имя]"
        echo ""
        echo "  add <имя>     — создать нового клиента"
        echo "  remove <имя>  — удалить клиента"
        echo "  list          — список всех клиентов и статус подключения"
        echo "  show <имя>    — показать конфиг клиента"
        echo "  qr <имя>      — QR-код для Amnezia-приложения"
        ;;
esac