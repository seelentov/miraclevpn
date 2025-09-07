#!/bin/bash

go build cmd/main/main.go

ssh komkov.vv@10.8.0.1 'sudo systemctl stop api'

scp main komkov.vv@10.8.0.1:~/mii_api/api
scp .env komkov.vv@10.8.0.1:~/mii_api/.env

ssh komkov.vv@10.8.0.1 'sudo systemctl restart api'

go build cmd/admin/monitor.go
scp monitor komkov.vv@10.8.0.1:~/mii_api/monitor 

ssh komkov.vv@10.8.0.1 'sudo journalctl -n 10000 -f -u api'