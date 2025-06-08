package client

import "github.com/mutablelogic/go-client/pkg/multipart"

/////////////////////////////////////////////////////////////////////////////////
// TYPES

type TranscribeRequest struct {
	Model          string         `json:"model_id"` // scribe_v1, scribe_v1_experimental
	File           multipart.File `json:"file"`
	Language       *string        `json:"language_code,omitempty"`
	TagAudioEvents *bool          `json:"tag_audio_events,omitempty"`
	NumSpeakers    *uint64        `json:"num_speakers,omitempty"`
	Timestamps     *string        `json:"timestamps_granularity,omitempty"` // none, word, character
	Diarize        *bool          `json:"diarize,omitempty"`
	FileFormat     *string        `json:"file_format,omitempty"` // pcm_s16le_16, other
}

type TranscribeResponse struct {
	Language    string  `json:"language_code"`
	Probability float64 `json:"language_probability"`
	Text        string  `json:"text"`
	Words       []struct {
		Text       string  `json:"text"`
		Type       string  `json:"type"`              // word, spacing, audio_event
		Logprob    float64 `json:"logprob,omitempty"` // -inf -> 0.0
		Start      float64 `json:"start"`
		End        float64 `json:"end"`
		Speaker    *string `json:"speaker,omitempty"`
		Characters struct {
			Text  string  `json:"text"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"characters,omitempty"` // Only present if timestamps_granularity is set to character
	} `json:"words,omitempty"` // Only present if timestamps_granularity is set to word or character
}

/////////////////////////////////////////////////////////////////////////////////
// GLOBALS

const (
	ElevenLabsEndpoint   = "https://api.elevenlabs.io/v1"
	ElevenLabsTranscribe = "speech-to-text" // Endpoint for transcription
)

var (
	ElevenLabsModels = []string{"scribe_v1", "scribe_v1_experimental"}
)
