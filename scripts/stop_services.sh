#!/bin/bash

sudo systemctl stop auth_suspicios.service & 
sudo systemctl stop healthcheck.service &
sudo systemctl stop server_priority.service &
sudo systemctl stop server_priority.service &
sudo systemctl stop vpn_refresh.service &
sudo systemctl stop payment_clear.service &
wait