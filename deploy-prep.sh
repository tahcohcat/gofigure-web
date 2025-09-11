#!/bin/bash

echo "🚀 GoFigure Web - Railway Deployment Preparation"
echo "================================================"
echo

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/server" ]; then
    echo "❌ Error: Please run this script from the gofigure-web project root directory"
    exit 1
fi

echo "✅ Project structure verified"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "⚠️  Warning: Docker not found. Railway will handle building, but you can't test locally."
else
    echo "✅ Docker found"
fi

# Check if git repo is initialized
if [ ! -d ".git" ]; then
    echo "📋 Initializing Git repository..."
    git init
    echo "✅ Git repository initialized"
else
    echo "✅ Git repository exists"
fi

# Check if files are committed
if [ -n "$(git status --porcelain)" ]; then
    echo "📋 Files need to be committed..."
    git add .
    git commit -m "Prepare for Railway deployment
    
    - Add Dockerfile for containerization
    - Add railway.json configuration  
    - Add deployment documentation
    - Ready for Railway.app hosting"
    echo "✅ Files committed to git"
else
    echo "✅ All files are committed"
fi

echo
echo "🎯 Next Steps:"
echo "1. Push to GitHub: git push origin main"
echo "2. Go to Railway.app and connect your GitHub repo"
echo "3. Set environment variable: GOOGLE_APPLICATION_CREDENTIALS_JSON"
echo "4. Deploy automatically!"
echo
echo "💡 See DEPLOYMENT.md for detailed instructions"
echo "💰 Expected cost: $0-8/month for personal use"
echo

# Check if we can test build locally
if command -v docker &> /dev/null; then
    echo "🧪 Want to test the Docker build locally? (y/n)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo "🔨 Building Docker image..."
        if docker build -t gofigure-web-test .; then
            echo "✅ Docker build successful!"
            echo "💡 You can test locally with: docker run -p 8080:8080 gofigure-web-test"
        else
            echo "❌ Docker build failed. Check the error messages above."
        fi
    fi
fi

echo
echo "🚀 Ready for Railway deployment!"