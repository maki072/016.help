#!/bin/bash
# Quick update script: pull from GitHub, rebuild, restart
set -e

APP_DIR=/opt/helpdesk
SRC_DIR=$APP_DIR/src

echo "=== Updating helpdesk ==="

cd $SRC_DIR
git pull origin main

echo "Building..."
export PATH=$PATH:/usr/local/go/bin
go build -o $APP_DIR/helpdesk main.go

echo "Copying templates and migrations..."
cp -r templates $APP_DIR/
cp -r migrations $APP_DIR/

echo "Restarting service..."
systemctl restart helpdesk
sleep 2
systemctl status helpdesk --no-pager | tail -8

echo "=== Done ==="
