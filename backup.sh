#!/usr/bin/env bash

echo docker stopping...
docker stop success-bot_bot_1 success-bot_db_1

rm -rf /home/bot/repo/success-bot/.backups/bot_db_bkp.tar.gz

echo archivating...
mkdir -p /home/bot/repo/success-bot/.backups
tar -czf "/home/bot/repo/success-bot/.backups/bot_db_bkp.tar.gz" "/home/bot/repo/success-bot/.database"

echo docker-compose starting...
cd /home/bot/repo/success-bot && docker-compose up -d