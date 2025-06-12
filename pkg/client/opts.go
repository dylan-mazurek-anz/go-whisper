package client

import (
	"io"
	"slices"

	// Packages
	"github.com/mutablelogic/go-client/pkg/multipart"
	"github.com/mutablelogic/go-server/pkg/httpresponse"
	"github.com/mutablelogic/go-server/pkg/types"
	"github.com/mutablelogic/go-whisper/pkg/client/elevenlabs"
	"github.com/mutablelogic/go-whisper/pkg/client/openai"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

// Request options
type opts struct {
	openai.TranscriptionRequest
	elevenlabs.TranscribeRequest
}

type Opt func(apitype, *opts) error

type apitype uint

///////////////////////////////////////////////////////////////////////////////
// GLOBALS

const (
	apiopenai apitype = iota
	apielevenlabs
)

///////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

func applyOpts(api apitype, model string, r io.Reader, opt ...Opt) (*opts, error) {
	var o opts
	o.Model = model
	o.TranscriptionRequest.File = multipart.File{Body: r}
	o.TranscribeRequest.File = multipart.File{Body: r}
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
		case apiopenai:
			// OpenAI uses two-letter language codes
			if code, _ := LanguageCode(language); code == "" {
				return httpresponse.ErrBadRequest.Withf("language %q not supported", language)
			} else {
				o.TranscriptionRequest.Language = types.StringPtr(code)
			}
		case apielevenlabs:
			// ElevenLabs uses three-letter language codes
			if _, code := LanguageCode(language); code == "" {
				return httpresponse.ErrBadRequest.Withf("language %q not supported", language)
			} else {
				o.TranscribeRequest.Language = types.StringPtr(language)
			}
		default:
			return httpresponse.ErrBadRequest.Withf("invalid API type %d", api)
		}
		return nil
	}
}

// Set format for transcription (json, srt, vtt, text)
func OptFormat(v string) Opt {
	return func(api apitype, o *opts) error {
		// Convert json to verbose format
		if v == "json" {
			v = openai.FormatJson
		}
		o.TranscriptionRequest.Format = types.StringPtr(v)
		return nil
	}
}

// Set path for the file to be transcribed
func OptPath(v string) Opt {
	return func(api apitype, o *opts) error {
		o.TranscriptionRequest.File.Path = v
		o.TranscribeRequest.File.Path = v
		return nil
	}
}

// Text to guide the model's style or continue a previous audio segment.
func OptPrompt(v string) Opt {
	return func(api apitype, o *opts) error {
		o.TranscriptionRequest.Prompt = types.StringPtr(v)
		return nil
	}
}

// The sampling temperature, between 0 and 1.
func OptTemperature(v float64) Opt {
	return func(api apitype, o *opts) error {
		o.TranscriptionRequest.Temperature = types.Float64Ptr(v)
		return nil
	}
}

// Return the log probabilities of the tokens in the response to understand the model's confidence in the transcription.
func OptLogprobs() Opt {
	return func(api apitype, o *opts) error {
		if !slices.Contains(o.TranscriptionRequest.Include, "logprobs") {
			o.TranscriptionRequest.Include = append(o.TranscriptionRequest.Include, "logprobs")
		}
		return nil
	}
}

// Model response data will be streamed to the client as it is generated using server-sent events.
func OptStream() Opt {
	return func(api apitype, o *opts) error {
		o.TranscriptionRequest.Stream = types.BoolPtr(true)
		return nil
	}
}

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

// Character-level timestamp granularities to populate for this transcription.
func OptDiarize() Opt {
	return func(api apitype, o *opts) error {
		switch api {
		case apielevenlabs:
			o.TranscribeRequest.Diarize = types.BoolPtr(true)
		default:
			return httpresponse.ErrBadRequest.With("diarization not supported")
		}
		return nil
	}
}
