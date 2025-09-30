#!/bin/sh

go build -o bin/monitor cmd/admin/monitor.go &
go build -o bin/auth_suspicios cmd/daemon/auth/suspicios.go &
go build -o bin/healthcheck cmd/daemon/healthcheck/healthcheck.go &
go build -o bin/server_priority cmd/daemon/server/priority.go &
go build -o bin/vpn_refresh cmd/daemon/vpn/refresh.go &
go build -o bin/payment cmd/daemon/payment/payment.go &
wait