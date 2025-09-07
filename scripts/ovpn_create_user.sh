#!/bin/bash

# Проверка root
if [[ "$EUID" -ne 0 ]]; then
    echo "This script needs to be run with superuser privileges."
    exit 1
fi

# Проверка что OpenVPN установлен
if [[ ! -e /etc/openvpn/server/server.conf ]]; then
    echo "OpenVPN is not installed. Please install it first."
    exit 1
fi

# Проверка аргументов
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 username [username2 ...]"
    exit 1
fi

# Определяем group_name
if grep -qs "ubuntu" /etc/os-release || [[ -e /etc/debian_version ]]; then
    group_name="nogroup"
else
    group_name="nobody"
fi

# Обрабатываем каждого пользователя
for client in "$@"; do
    # Проверяем имя пользователя
    unsanitized_client="$client"
    client=$(sed 's/[^0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-]/_/g' <<< "$unsanitized_client")
    if [[ -z "$client" ]]; then
        echo "Invalid username: $unsanitized_client"
        continue
    fi
    
    # Проверяем существует ли уже пользователь
    if [[ -e /etc/openvpn/server/easy-rsa/pki/issued/"$client".crt ]]; then
        echo "Client $client already exists."
        continue
    fi
    
    # Создаем сертификат
    cd /etc/openvpn/server/easy-rsa/
    ./easyrsa --batch --days=3650 build-client-full "$client" nopass
    
    # Генерируем конфиг
    {
    cat /etc/openvpn/server/client-common.txt
    echo "<ca>"
    cat /etc/openvpn/server/easy-rsa/pki/ca.crt
    echo "</ca>"
    echo "<cert>"
    sed -ne '/BEGIN CERTIFICATE/,$ p' /etc/openvpn/server/easy-rsa/pki/issued/"$client".crt
    echo "</cert>"
    echo "<key>"
    cat /etc/openvpn/server/easy-rsa/pki/private/"$client".key
    echo "</key>"
    echo "<tls-crypt>"
    sed  -ne '/BEGIN OpenVPN Static key/,$ p' /etc/openvpn/server/tc.key
    echo "</tls-crypt>"
    } > "$(pwd)/$client.ovpn"
    
    echo "Client $client created. Configuration available at: $(pwd)/$client.ovpn"
    cat $(pwd)/$client.ovpn
done
