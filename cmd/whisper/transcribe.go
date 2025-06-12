package main

import (
	"bytes"
	"os"
	"time"

	// Packages
	goclient "github.com/mutablelogic/go-client"
	segmenter "github.com/mutablelogic/go-media/pkg/segmenter"
	httpresponse "github.com/mutablelogic/go-server/pkg/httpresponse"
	whisper "github.com/mutablelogic/go-whisper"
	client "github.com/mutablelogic/go-whisper/pkg/client"
	schema "github.com/mutablelogic/go-whisper/pkg/schema"
	task "github.com/mutablelogic/go-whisper/pkg/task"
	wav "github.com/mutablelogic/go-whisper/pkg/wav"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type TranslateCmd struct {
	Model    string        `arg:"" help:"Model to use"`
	Path     string        `arg:"" help:"Path to audio file"`
	Format   string        `flag:"" help:"Output format" default:"text" enum:"json,verbose_json,text,vtt,srt"`
	Segments time.Duration `flag:"" help:"Segment size for reading audio file"`
	Api      bool          `flag:"" help:"Use API for translation or transcription"`
	Diarize  bool          `flag:"" help:"Diarize the transcription"`
}

type TranscribeCmd struct {
	TranslateCmd
	Language string `flag:"language" help:"Language to transcribe"`
}

////////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func (cmd *TranscribeCmd) Run(app *Globals) error {
	if cmd.Api {
		return run_remote(app, cmd.Model, cmd.Path, cmd.Language, cmd.Format, cmd.Segments, false, cmd.Diarize)
	} else {
		return run_local(app, cmd.Model, cmd.Path, cmd.Language, cmd.Format, cmd.Segments, false)
	}
}

func (cmd *TranslateCmd) Run(app *Globals) error {
	if cmd.Api {
		return run_remote(app, cmd.Model, cmd.Path, "", cmd.Format, cmd.Segments, true, cmd.Diarize)
	} else {
		return run_local(app, cmd.Model, cmd.Path, "", cmd.Format, cmd.Segments, true)
	}
}

func run_local(app *Globals, model, path, language, format string, segments time.Duration, translate bool) error {
	// Get the model
	model_ := app.service.GetModelById(model)
	if model_ == nil {
		return httpresponse.ErrNotFound.With(model)
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

func run_remote(app *Globals, model, path, language, format string, segments time.Duration, translate, diarize bool) error {
	// Open the audio file
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a client for the whisper service
	opts := []goclient.ClientOpt{
		goclient.OptTimeout(5 * time.Minute), // Set a timeout for the request
	}
	if app.Debug {
		opts = append(opts, goclient.OptTrace(os.Stderr, false))
	}
	remote, err := client.New(opts...)
	if err != nil {
		return err
	}

	// Create an array of parameters for the transcription
	params := []client.Opt{
		client.OptPath("audio.wav"), client.OptFormat("json"),
	}
	if !translate && language != "" {
		params = append(params, client.OptLanguage(language))
	}
	if diarize {
		params = append(params, client.OptDiarize())
	}

	// Create a segmenter - read segments based on requested segment size
	segmenter, err := segmenter.NewReader(f, segments, whisper.SampleRate)
	if err != nil {
		return err
	}
	defer segmenter.Close()

	// Read samples and transcribe or translate them
	return segmenter.DecodeInt16(app.ctx, func(ts time.Duration, data []int16) error {
		// Make a WAV file from the float32 samples
		r, err := wav.NewInt16(data, whisper.SampleRate)
		if err != nil {
			return err
		}

		var segments []*schema.Segment
		if translate {
			translation, err := remote.Translate(app.ctx, model, r, params...)
			if err != nil {
				return err
			} else {
				segments = translation.Segments
			}
		} else {
			transcription, err := remote.Transcribe(app.ctx, model, r, params...)
			if err != nil {
				return err
			} else {
				segments = transcription.Segments
			}
		}

		// Write the segments to the writer
		for _, segment := range segments {
			var buf bytes.Buffer
			switch format {
			case "json", "verbose_json":
				app.writer.Writeln(segment)
			case "srt":
				segment.WriteSRT(&buf, ts)
				app.writer.Writeln(buf.String())
			case "vtt":
				segment.WriteVTT(&buf, ts)
				app.writer.Writeln(buf.String())
			case "text":
				segment.WriteText(&buf)
				app.writer.Writeln(buf.String())
			}
		}

		return nil
	})
}
