#!/bin/bash

sudo systemctl start auth_suspicios.service & 
sudo systemctl start healthcheck.service &
sudo systemctl start server_priority.service &
sudo systemctl start server_priority.service &
sudo systemctl start vpn_refresh.service &
wait