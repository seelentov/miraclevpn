#!/bin/bash

# Функция для выполнения команд на удаленном сервере
remote_exec() {
    ssh komkov.vv@10.8.0.1 "$@"
}

# Функция для копирования файлов
copy_files() {
    local local_file=$1
    local remote_path=$2
    local filename=$(basename "$local_file")
    
    # Копируем во временную директорию
    scp "$local_file" "komkov.vv@10.8.0.1:/tmp/$filename"
    # Перемещаем с sudo правами
    remote_exec "sudo mv /tmp/$filename $remote_path/$filename"
}

# Функция для перезапуска сервиса с проверкой
restart_service() {
    local service=$1
    echo "Перезапуск сервиса $service"
    remote_exec "sudo systemctl restart $service"
}

# 1. Копируем systemd файлы и создаем директории
echo "Копирование systemd файлов..."
copy_files "systemd/*" "/etc/systemd/system/"
remote_exec "mkdir -p ~/mii_api"

# 2. Собираем и деплоим API как можно быстрее
echo "Сборка и деплой API..."
go build -o main cmd/main/main.go

# Останавливаем API только когда все готово для быстрого перезапуска
remote_exec 'sudo systemctl stop api'

# Параллельное копирование основных файлов API
copy_files "main" "~/mii_api/api" &
copy_files ".env" "~/mii_api/.env" &
wait

# Немедленный перезапуск API
restart_service "api"

# 3. Параллельная сборка и деплой остальных компонентов
echo "Сборка и деплой дополнительных компонентов..."

# Сборка всех компонентов параллельно
go build -o monitor cmd/admin/monitor.go &
go build -o auth_suspicios cmd/daemon/auth/suspicios.go &
go build -o healthcheck cmd/daemon/healthcheck/healthcheck.go &
go build -o server_priority cmd/daemon/server/priority.go &
go build -o vpn_refresh cmd/daemon/vpn/refresh.go &
wait

# Деплой компонентов с минимальным временем простоя
(
    remote_exec 'sudo systemctl stop auth_suspicios'
    copy_files "auth_suspicios" "~/mii_api/daemon/auth_suspicios"
    restart_service "auth_suspicios"
) &

(
    remote_exec 'sudo systemctl stop healthcheck'
    copy_files "healthcheck" "~/mii_api/daemon/healthcheck"
    restart_service "healthcheck"
) &

(
    remote_exec 'sudo systemctl stop server_priority'
    copy_files "server_priority" "~/mii_api/daemon/server_priority"
    restart_service "server_priority"
) &

(
    remote_exec 'sudo systemctl stop vpn_refresh'
    copy_files "vpn_refresh" "~/mii_api/daemon/vpn_refresh"
    restart_service "vpn_refresh"
) &

# Копирование monitor отдельно
copy_files "monitor" "~/mii_api/monitor" &

wait

# 4. Включение автозапуска сервисов
echo "Включение автозапуска сервисов..."
remote_exec 'sudo systemctl enable api auth_suspicios healthcheck server_priority vpn_refresh'

# 5. Мониторинг логов API
echo "Запуск мониторинга логов API..."
remote_exec 'sudo journalctl -n 100 -f -u api'