package api

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	// Packages
	"github.com/mutablelogic/go-media/pkg/segmenter"
	"github.com/mutablelogic/go-server/pkg/httprequest"
	"github.com/mutablelogic/go-server/pkg/httpresponse"
	"github.com/mutablelogic/go-server/pkg/types"
	"github.com/mutablelogic/go-whisper"
	"github.com/mutablelogic/go-whisper/pkg/client/gowhisper"
	"github.com/mutablelogic/go-whisper/pkg/schema"
	"github.com/mutablelogic/go-whisper/pkg/task"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

type reqTranscribe struct {
	File        *multipart.FileHeader `json:"file"`
	Model       string                `json:"model"`
	Language    *string               `json:"language"`
	Temperature *float32              `json:"temperature"`
	SegmentSize *time.Duration        `json:"segment_size"`
	ResponseFmt *string               `json:"response_format"`
}

type queryTranscribe struct {
	Stream bool `json:"stream"`
}

type TaskType int
type ResponseFormat string

///////////////////////////////////////////////////////////////////////////////
// GLOBALS

const (
	minSegmentSize     = 5 * time.Second
	maxSegmentSize     = 10 * time.Minute
	defaultSegmentSize = 5 * time.Minute
)

const (
	FormatJson        ResponseFormat = "json"
	FormatText        ResponseFormat = "text"
	FormatSrt         ResponseFormat = "srt"
	FormatVerboseJson ResponseFormat = "verbose_json"
	FormatVtt         ResponseFormat = "vtt"
)

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func TranscribeFile(ctx context.Context, service *whisper.Whisper, w http.ResponseWriter, r *http.Request) error {
	// Read the request
	var req gowhisper.TranscriptionRequest
	if err := httprequest.Read(r, &req); err != nil {
		return httpresponse.Error(w, httpresponse.ErrBadRequest, err.Error())
	}

	// Get the model
	model := service.GetModelById(req.Model)
	if model == nil {
		return httpresponse.Error(w, httpresponse.ErrNotFound, req.Model)
	}

	// Start a translation task
	var response *schema.Transcription
	if err := service.WithModel(model, func(taskctx *task.Context) error {
		taskctx.SetTranslate(false)
		taskctx.SetDiarize(types.PtrBool(req.Diarize))

		// Set language
		lang := types.PtrString(req.Language)
		if lang == "" {
			lang = "auto"
		}
		if err := taskctx.SetLanguage(lang); err != nil {
			return err
		}

		// Set temperature
		if req.Temperature != nil {
			if err := taskctx.SetTemperature(types.PtrFloat64(req.Temperature)); err != nil {
				return err
			}
		}

		// Set prompt
		if req.Prompt != nil {
			if err := taskctx.SetPrompt(types.PtrString(req.Prompt)); err != nil {
				return err
			}
		}

		// Set response
		response = taskctx.Result()

		// Decode, resample and segment the audio file
		return segment(ctx, w, taskctx, req.File.Body, func(seg *schema.Segment) {
			// TODO - for streaming
		})
	}); err != nil {
		return httpresponse.Error(w, httpresponse.ErrInternalError, err.Error())
	}

	// Return the original request
	return httpresponse.JSON(w, http.StatusOK, 2, response)
}

func TranslateFile(ctx context.Context, service *whisper.Whisper, w http.ResponseWriter, r *http.Request) error {
	// Read the request
	var req gowhisper.TranslationRequest
	if err := httprequest.Read(r, &req); err != nil {
		return httpresponse.Error(w, httpresponse.ErrBadRequest, err.Error())
	}

	// Get the model
	model := service.GetModelById(req.Model)
	if model == nil {
		return httpresponse.Error(w, httpresponse.ErrNotFound, req.Model)
	}

	// Return the original request
	return httpresponse.JSON(w, http.StatusOK, 2, req)
}

func segment(ctx context.Context, w http.ResponseWriter, taskctx *task.Context, r io.Reader, fn func(seg *schema.Segment)) error {
	// Create a segmenter
	segmenter, err := segmenter.NewReader(r, 0, whisper.SampleRate)
	if err != nil {
		return err
	}

	// Read segments and perform transcription
	if err := segmenter.DecodeFloat32(ctx, func(ts time.Duration, buf []float32) error {
		return taskctx.Transcribe(ctx, ts, buf, fn)
	}); err != nil {
		return err
	}

	// Return sucess
	return nil
}

/*


	// Create a text stream
	var stream *httpresponse.TextStream
	if query.Stream {
		if stream = httpresponse.NewTextStream(w); stream == nil {
			httpresponse.Error(w, httpresponse.ErrInternalError, "Cannot create text stream")
			return
		}
		defer stream.Close()
	}

	// Get context for the model, perform transcription
	var result *schema.Transcription
	if err := service.WithModel(model, func(taskctx *task.Context) error {
		result = taskctx.Result()

		switch t {
		case Translate:
			// Check model
			if !taskctx.CanTranslate() {
				return ErrBadParameter.With("model is not multilingual, cannot translate")
			}
			taskctx.SetTranslate(true)
			taskctx.SetDiarize(false)
			result.Task = "translate"

			// Set language to EN
			if err := taskctx.SetLanguage("en"); err != nil {
				return err
			}
		case Diarize:
			taskctx.SetTranslate(false)
			taskctx.SetDiarize(true)
			result.Task = "diarize"

			// Set language
			if req.Language != nil {
				if err := taskctx.SetLanguage(*req.Language); err != nil {
					return err
				}
			}
		default:
			// Transcribe
			taskctx.SetTranslate(false)
			taskctx.SetDiarize(false)
			result.Task = "transribe"

			// Set language
			if req.Language != nil {
				if err := taskctx.SetLanguage(*req.Language); err != nil {
					return err
				}
			}
		}

		// TODO: Set temperature, etc

		// Output the header
		result.Language = taskctx.Language()
		if stream != nil {
			stream.Write("task", taskctx.Result())
		}

		// Read samples and transcribe them
		if err := segmenter.DecodeFloat32(ctx, func(ts time.Duration, buf []float32) error {
			// Perform the transcription, return any errors
			return taskctx.Transcribe(ctx, ts, buf, func(segment *schema.Segment) {
				// Segment callback
				if stream == nil {
					return
				}
				var buf bytes.Buffer
				switch req.ResponseFormat() {
				case FormatVerboseJson, FormatJson:
					stream.Write("segment", segment)
					return
				case FormatSrt:
					task.WriteSegmentSrt(&buf, segment)
				case FormatVtt:
					task.WriteSegmentVtt(&buf, segment)
				case FormatText:
					task.WriteSegmentText(&buf, segment)
				}
				stream.Write("segment", buf.String())
			})
		}); err != nil {
			return err
		}

		// Set the language and duration
		result.Language = taskctx.Language()

		// Return success
		return nil
	}); err != nil {
		if stream != nil {
			stream.Write("error", err.Error())
		} else {
			httpresponse.Error(w, httpresponse.ErrInternalError, err.Error())
		}
		return
	}

	// Return transcription if not streaming
	if stream == nil {
		httpresponse.JSON(w, http.StatusOK, 2, result)
	} else {
		stream.Write("ok")
	}
*/

/*
func TranscribeStream(ctx context.Context, service *whisper.Whisper, w http.ResponseWriter, r *http.Request, modelId string) {
	var query queryTranscribe
	if err := httprequest.Query(&query, r.URL.Query()); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get the model
	model := service.GetModelById(modelId)
	if model == nil {
		httpresponse.Error(w, http.StatusNotFound, "model not found")
		return
	}

	// Create a segmenter - read segments based on 10 second segment size
	segmenter, err := segmenter.New(r.Body, 10*time.Second, whisper.SampleRate)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create a text stream
	var stream *httpresponse.TextStream
	if query.Stream {
		if stream = httpresponse.NewTextStream(w); stream == nil {
			httpresponse.Error(w, http.StatusInternalServerError, "Cannot create text stream")
			return
		}
		defer stream.Close()
	}

	// Get context for the model, perform transcription
	var result *schema.Transcription
	if err := service.WithModel(model, func(task *task.Context) error {
		// Set parameters for ttranslation, default to auto
		task.SetTranslate(false)
		if err := task.SetLanguage("auto"); err != nil {
			return err
		}

		// TODO: Set temperature, etc

		// Create response
		result = task.Result()
		result.Task = "transcribe"
		result.Language = task.Language()

		// Output the header
		if stream != nil {
			stream.Write("task", result)
		}

		// Read samples and transcribe them
		if err := segmenter.Decode(ctx, func(ts time.Duration, buf []float32) error {
			// Perform the transcription, output segments in realtime, return any errors
			return task.Transcribe(ctx, ts, buf, func(segment *schema.Segment) {
				if stream != nil {
					stream.Write("segment", segment)
				}
			})
		}); err != nil {
			return err
		}

		// Set the language
		result.Language = taskctx.Language()

		// Return success
		return nil
	}); err != nil {
		if stream != nil {
			stream.Write("error", err.Error())
		} else {
			httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return streaming ok
	if stream != nil {
		stream.Write("ok")
		return
	}

	// Rrturn result based on response format
	switch req.ResponseFormat() {
	case FormatJson, FormatVerboseJson:
		httpresponse.JSON(w, result, http.StatusOK, 0)
	case FormatText:
		httpresponse.Text(w, "", http.StatusOK)
		for _, seg := range result.Segments {
			task.WriteSegmentText(w, seg)
		}
		w.Write([]byte("\n"))
	case FormatSrt:
		httpresponse.Text(w, "", http.StatusOK, "Content-Type", "application/x-subrip")
		for _, seg := range result.Segments {
			task.WriteSegmentSrt(w, seg)
		}
	case FormatVtt:
		httpresponse.Text(w, "WEBVTT\n\n", http.StatusOK, "Content-Type", "text/vtt")
		for _, seg := range result.Segments {
			task.WriteSegmentVtt(w, seg)
		}
	}
}
*/

///////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (r reqTranscribe) Validate() error {
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}
	if r.File == nil {
		return fmt.Errorf("file is required")
	}
	if r.ResponseFmt != nil {
		switch *r.ResponseFmt {
		case "json", "text", "srt", "verbose_json", "vtt":
			break
		default:
			return fmt.Errorf("response_format must be one of: json, text, srt, verbose_json, vtt")
		}
	}
	return nil
}
func (r reqTranscribe) ResponseFormat() ResponseFormat {
	if r.ResponseFmt == nil {
		return FormatJson
	}
	switch strings.ToLower(*r.ResponseFmt) {
	case "json":
		return FormatJson
	case "text":
		return FormatText
	case "srt":
		return FormatSrt
	case "verbose_json":
		return FormatVerboseJson
	case "vtt":
		return FormatVtt
	}
	return FormatJson
}

func (r reqTranscribe) OutputSegments() bool {
	// We want to output segments if the response format is  "srt", "verbose_json", "vtt"
	switch r.ResponseFormat() {
	case FormatSrt, FormatVerboseJson, FormatVtt:
		return true
	default:
		return false
	}
}

func (r reqTranscribe) SegmentDur() time.Duration {
	if r.SegmentSize == nil {
		return defaultSegmentSize
	}
	if *r.SegmentSize < minSegmentSize {
		return minSegmentSize
	}
	if *r.SegmentSize > maxSegmentSize {
		return maxSegmentSize
	}
	return *r.SegmentSize
}
