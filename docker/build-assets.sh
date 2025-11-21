#!/bin/bash
set -e

echo "Building web assets..."

# Check if node_modules exists, if not install dependencies
if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
fi

# Build minified CSS and JS
echo "Minifying CSS..."
npm run build:css

echo "Minifying JavaScript..."
npm run build:js

echo "Asset build complete!"
echo "  - web/static/style.min.css"
echo "  - web/static/app.min.js"
