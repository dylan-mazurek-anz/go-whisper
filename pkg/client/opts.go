package client

import (
	"io"
	"slices"

	// Packages
	"github.com/mutablelogic/go-client/pkg/multipart"
	"github.com/mutablelogic/go-server/pkg/httpresponse"
	"github.com/mutablelogic/go-server/pkg/types"
	"github.com/mutablelogic/go-whisper/pkg/client/elevenlabs"
	"github.com/mutablelogic/go-whisper/pkg/client/gowhisper"
	"github.com/mutablelogic/go-whisper/pkg/client/openai"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

// Request options
type opts struct {
	openai     openai.TranscriptionRequest
	elevenlabs elevenlabs.TranscribeRequest
	gowhisper  gowhisper.TranscriptionRequest
}

type Opt func(apitype, *opts) error

type apitype uint

///////////////////////////////////////////////////////////////////////////////
// GLOBALS

const (
	apiopenai apitype = iota
	apielevenlabs
	apigowhisper
)

///////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

func applyOpts(api apitype, model string, r io.Reader, opt ...Opt) (*opts, error) {
	var o opts

	o.openai.File = multipart.File{Body: r}
	o.openai.Model = model
	o.elevenlabs.File = multipart.File{Body: r}
	o.elevenlabs.Model = model
	o.gowhisper.File = multipart.File{Body: r}
	o.gowhisper.Model = model

	for _, opt := range opt {
		if err := opt(api, &o); err != nil {
			return nil, err
		}
	}
	return &o, nil
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

// Set language for transcription
func OptLanguage(language string) Opt {
	return func(api apitype, o *opts) error {
		if language == "" {
			return nil
		}
		switch api {
		case apiopenai, apigowhisper:
			// whisper uses two-letter language codes
			if code, _ := LanguageCode(language); code == "" {
				return httpresponse.ErrBadRequest.Withf("language %q not supported", language)
			} else {
				o.openai.Language = types.StringPtr(code)
				o.gowhisper.Language = types.StringPtr(code)
			}
		case apielevenlabs:
			// ElevenLabs uses three-letter language codes
			if _, code := LanguageCode(language); code == "" {
				return httpresponse.ErrBadRequest.Withf("language %q not supported", language)
			} else {
				o.elevenlabs.Language = types.StringPtr(language)
			}
		default:
			return httpresponse.ErrBadRequest.Withf("invalid API type %d", api)
		}
		return nil
	}
}

// Set format for transcription (json, verbose_json, srt, vtt, text)
func OptFormat(v string) Opt {
	return func(api apitype, o *opts) error {
		// Check format
		if !slices.Contains(openai.Formats, v) {
			return httpresponse.ErrBadRequest.Withf("format %q not supported", v)
		}

		// Set format
		switch api {
		case apiopenai, apigowhisper:
			o.gowhisper.Format = types.StringPtr(v)
			o.openai.Format = types.StringPtr(v)
		default:
			return httpresponse.ErrBadRequest.Withf("format %q not supported", v)
		}

		// Return success
		return nil
	}
}

// Set path for the file to be transcribed
func OptPath(v string) Opt {
	return func(api apitype, o *opts) error {
		o.openai.File.Path = v
		o.elevenlabs.File.Path = v
		o.gowhisper.File.Path = v
		return nil
	}
}

// Text to guide the model's style or continue a previous audio segment.
func OptPrompt(v string) Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apiopenai, apigowhisper:
			o.openai.Prompt = types.StringPtr(v)
			o.gowhisper.Prompt = types.StringPtr(v)
		default:
			return httpresponse.ErrNotImplemented.Withf("OptPrompt not supported")
		}
		return nil
	}
}

// The sampling temperature, between 0 and 1.
func OptTemperature(v float64) Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apiopenai, apigowhisper:
			o.openai.Temperature = types.Float64Ptr(v)
			o.gowhisper.Temperature = types.Float64Ptr(v)
		default:
			return httpresponse.ErrNotImplemented.Withf("OptTemperature not supported")
		}
		return nil
	}
}

// Return the log probabilities of the tokens in the response to understand the model's confidence in the transcription.
func OptLogprobs() Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apiopenai:
			if !slices.Contains(o.openai.Include, "logprobs") {
				o.openai.Include = append(o.openai.Include, "logprobs")
			}
		default:
			return httpresponse.ErrNotImplemented.Withf("OptLogprobs not supported")
		}
		return nil
	}
}

// Model response data will be streamed to the client as it is generated using server-sent events.
func OptStream() Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apiopenai, apigowhisper:
			o.openai.Stream = types.BoolPtr(true)
			o.gowhisper.Stream = types.BoolPtr(true)
		default:
			return httpresponse.ErrNotImplemented.Withf("OptStream not supported")
		}
		return nil
	}
}

// Identify speakers in the audio and return their speech separately.
func OptDiarize() Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apigowhisper:
			o.gowhisper.Diarize = types.BoolPtr(true)
		case apielevenlabs:
			o.elevenlabs.Diarize = types.BoolPtr(true)
		default:
			return httpresponse.ErrBadRequest.With("diarization not supported")
		}
		return nil
	}
}

/*
// Word-level timestamp granularities to populate for this transcription.
func OptGranularityWord() Opt {
	return func(api apitype, o *opts) error {
		o.TranscribeRequest.Timestamps = types.StringPtr("word")
		if !slices.Contains(o.TranscriptionRequest.Timestamps, "word") {
			o.TranscriptionRequest.Include = append(o.TranscriptionRequest.Timestamps, "word")
		}
		return nil
	}
}

// Character-level timestamp granularities to populate for this transcription.
func OptGranularityChar() Opt {
	return func(api apitype, o *opts) error {
		o.TranscribeRequest.Timestamps = types.StringPtr("character")
		return nil
	}
}

// Segment-level timestamp granularities to populate for this transcription.
func OptGranularitySegment() Opt {
	return func(api apitype, o *opts) error {
		if !slices.Contains(o.TranscriptionRequest.Timestamps, "segment") {
			o.TranscriptionRequest.Include = append(o.TranscriptionRequest.Timestamps, "segment")
		}
		return nil
	}
}

*/
