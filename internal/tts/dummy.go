package tts

import (
	"context"
	"github.com/tahcohcat/gofigure-web/internal/logger"
)

type DummyTts struct {
}

func NewDummyTts() *DummyTts {
	return &DummyTts{}
}

func (d *DummyTts) Speak(_ context.Context, text, emotion string, model TTSModel) error {
	logger.New().Debug("no tts configured. ignoring TTS request")
	return nil
}

func (d *DummyTts) Name() string {
	return "dummy"
}
