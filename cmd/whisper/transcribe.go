package main

import (
	"bytes"
	"os"
	"time"

	// Packages
	segmenter "github.com/mutablelogic/go-media/pkg/segmenter"
	whisper "github.com/mutablelogic/go-whisper"
	schema "github.com/mutablelogic/go-whisper/pkg/schema"
	task "github.com/mutablelogic/go-whisper/pkg/task"

	// Namespace imports
	. "github.com/djthorpe/go-errors"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type TranslateCmd struct {
	Model    string        `arg:"" help:"Model to use"`
	Path     string        `arg:"" help:"Path to audio file"`
	Format   string        `flag:"" help:"Output format" default:"text" enum:"json,verbose_json,text,vtt,srt"`
	Segments time.Duration `flag:"" help:"Segment size for reading audio file"`
}

type TranscribeCmd struct {
	TranslateCmd
	Language string `flag:"language" help:"Language to transcribe" default:"auto"`
}

////////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func (cmd *TranscribeCmd) Run(app *Globals) error {
	return run(app, cmd.Model, cmd.Path, cmd.Language, cmd.Format, cmd.Segments, false)
}

func (cmd *TranslateCmd) Run(app *Globals) error {
	return run(app, cmd.Model, cmd.Path, "", cmd.Format, cmd.Segments, true)
}

func run(app *Globals, model, path, language, format string, segments time.Duration, translate bool) error {
	// Get the model
	model_ := app.service.GetModelById(model)
	if model_ == nil {
		return ErrNotFound.With(model)
	}

	// Open the audio file
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a segmenter - read segments based on requested segment size
	segmenter, err := segmenter.NewReader(f, segments, whisper.SampleRate)
	if err != nil {
		return err
	}
	defer segmenter.Close()

	// Perform the transcription
	return app.service.WithModel(model_, func(taskctx *task.Context) error {
		// Transcribe or Translate
		taskctx.SetTranslate(translate)
		taskctx.SetDiarize(false)

		// Set language
		if language != "" {
			if err := taskctx.SetLanguage(language); err != nil {
				return err
			}
		}

		// Read samples and transcribe them
		if err := segmenter.DecodeFloat32(app.ctx, func(ts time.Duration, buf []float32) error {
			// Perform the transcription, return any errors
			return taskctx.Transcribe(app.ctx, ts, buf, func(segment *schema.Segment) {
				var buf bytes.Buffer
				switch format {
				case "json", "verbose_json":
					app.writer.Writeln(segment)
				case "srt":
					task.WriteSegmentSrt(&buf, segment)
					app.writer.Writeln(buf.String())
				case "vtt":
					task.WriteSegmentVtt(&buf, segment)
					app.writer.Writeln(buf.String())
				case "text":
					task.WriteSegmentText(&buf, segment)
					app.writer.Writeln(buf.String())
				}
			})
		}); err != nil {
			return err
		}

		return nil
	})
}
