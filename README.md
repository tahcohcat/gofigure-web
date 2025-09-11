# GoFigure Web

A web-based murder mystery roleplay game powered by AI.

## Quick Start

1. **Install dependencies:**
```bash
go mod tidy
```

2. **Set up environment variables:**
```bash
export OPENAI_API_KEY="your-openai-api-key-here"
```

3. **Run the server:**
```bash
go run cmd/server/main.go
```

4. **Open your browser:**
Navigate to `http://localhost:8080`

## Project Structure

```
├── cmd/server/          # Web server entry point
├── internal/
│   ├── api/            # REST API handlers
│   ├── websocket/      # WebSocket handlers
│   ├── game/           # Core game logic
│   ├── llm/            # LLM integrations (OpenAI, etc.)
│   └── logger/         # Logging utilities
├── web/
│   ├── static/         # CSS, JS, images
│   └── templates/      # HTML templates
├── config/             # Configuration management
├── data/               # Mystery JSON files
└── config.yaml         # Main configuration file
```

## Features

- 🎭 Interactive murder mystery gameplay
- 🤖 AI-powered character responses
- 🌐 Modern web interface
- 📱 Responsive design
- 🔄 Real-time communication via WebSockets

## Configuration

Edit `config.yaml` to customize:
- LLM provider (OpenAI)
- Model settings
- TTS/STT (currently disabled for web version)

## Adding Mysteries

Add new mystery JSON files to the `data/mysteries/` directory following the existing format.

## Development

The web version is designed to be self-contained and doesn't require the CLI version to run.

## 🚀 Deployment

### Deploy to Railway 

## 🎮 Mystery Scenarios

The app includes 4 built-in mysteries with different difficulty levels:

- 🟢 **Easy**: Secrets at Rosie's Diner (small-town mystery)
- 🟡 **Medium**: The Blackwood Manor Murder (classic manor house)
- 🟡 **Medium**: Corporate Betrayal (modern office intrigue)  
- 🔴 **Hard**: Death on the Aurora Star (complex cruise ship mystery)

## 🔊 Text-to-Speech

Features high-quality Google Chirp HD voices with:
- **Character-specific voices** from mystery JSON configuration
- **Emotion-based voice modulation** (pitch, speed, tone)
- **Centralized billing** (only your server calls Google TTS API)
- **Multiple language support** (en-US, en-GB, fr-FR, etc.)

## Development

The web version is designed to be self-contained and doesn't require the CLI version to run.