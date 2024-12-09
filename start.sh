#!/bin/bash
echo "Starting Nakama script"
./nakama --config local.yml > ./log/nakama.log 2>&1 &

echo "Nakama script finished"
