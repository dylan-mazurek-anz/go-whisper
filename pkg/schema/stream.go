package schema

import "encoding/json"

//////////////////////////////////////////////////////////////////////////////
// TYPES

type Event struct {
	Type  string `json:"type"`
	Delta string `json:"delta,omitempty"` // delta
	Text  string `json:"text,omitempty"`  // done
}

//////////////////////////////////////////////////////////////////////////////
// GLOBALS

const (
	DownloadStreamProgressType = "download.progress"
	DownloadStreamErrorType    = "download.error"
	DownloadStreamDoneType     = "download.done"
)

const (
	TranscribeStreamDeltaType    = "transcript.text.delta"
	TranscribeStreamDoneType     = "transcript.text.done"
	TranscribeStreamErrorType    = "transcript.text.error"
	TranscribeStreamLanguageType = "transcript.text.language"
)

//////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (e Event) String() string {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(data)
}
