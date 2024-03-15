#!/bin/sh

sh 'cd /home/ubuntu/sidekiq-server && git pull'
sh 'nohup /home/ubuntu/installations/go1.19/bin/go run main.go serve &'

sh 'exit'
