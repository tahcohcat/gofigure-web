[![Made with Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/)
[![Go Version](https://img.shields.io/badge/Go-1.24.3-blue.svg)](https://go.dev/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

[![OpenAI](https://img.shields.io/badge/AI-OpenAI-412991.svg)](https://openai.com) [![Google TTS](https://img.shields.io/badge/Speech-Google_TTS-4285F4.svg)](https://cloud.google.com/text-to-speech) [![Gorilla Mux](https://img.shields.io/badge/Router-Gorilla_Mux-7D4C9F.svg)](https://github.com/gorilla/mux) [![Viper](https://img.shields.io/badge/Config-Viper-3D3C78.svg)](https://github.com/spf13/viper)

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