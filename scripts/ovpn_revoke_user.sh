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
    # Проверяем существует ли пользователь
    if [[ ! -e /etc/openvpn/server/easy-rsa/pki/issued/"$client".crt ]]; then
        echo "Client $client does not exist."
        continue
    fi
    
    # Отзываем сертификат
    cd /etc/openvpn/server/easy-rsa/
    ./easyrsa --batch revoke "$client"
    ./easyrsa --batch --days=3650 gen-crl
    rm -f /etc/openvpn/server/crl.pem
    cp /etc/openvpn/server/easy-rsa/pki/crl.pem /etc/openvpn/server/crl.pem
    chown nobody:"$group_name" /etc/openvpn/server/crl.pem
    
    # Удаляем файлы пользователя
    rm -f /etc/openvpn/server/easy-rsa/pki/reqs/"$client".req
    rm -f /etc/openvpn/server/easy-rsa/pki/private/"$client".key
    rm -f /etc/openvpn/server/easy-rsa/pki/issued/"$client".crt
    
    echo "Client $client revoked!"
done
