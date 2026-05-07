#!/usr/bin/env bash

echo docker stopping...
docker stop success-bot_bot_1 success-bot_db_1 

#sleep 1m

echo archivating...
tar -czf "/home/bot/repo/success-bot/.backups/bot_db_bkp_$(date +%Y%m%d).tar.gz" "/home/bot/repo/success-bot/.database"

echo docker-compose starting...
cd /home/bot/repo/success-bot && docker-compose up -d