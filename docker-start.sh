#!/bin/bash

# Simple Docker startup script for URL Shortener

echo "🐳 Starting URL Shortener with Docker"

# Copy environment file if .env doesn't exist
if [ ! -f .env ]; then
    if [ -f .env.docker ]; then
        echo "📋 Copying .env.docker to .env"
        cp .env.docker .env
    else
        echo "⚠️  No .env.docker file found, using defaults"
    fi
else
    echo "✅ Using existing .env file"
fi

# Start Docker Compose
echo "🚀 Starting services..."
docker-compose up --build

echo "🎉 Done!"
