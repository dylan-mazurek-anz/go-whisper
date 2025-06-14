package api

import (
	"context"
	"io"
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
	var result *schema.Transcription
	if err := service.WithModel(model, func(taskctx *task.Context) error {
		taskctx.SetTranslate(false)
		taskctx.SetDiarize(types.PtrBool(req.Diarize))

		// Set language
		if err := taskctx.SetLanguage(types.PtrString(req.Language)); err != nil {
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
		result = taskctx.Result()

		// Decode, resample and segment the audio file
		return segment(ctx, w, taskctx, req.File.Body, func(seg *schema.Segment) {
			// TODO - for streaming
		})
	}); err != nil {
		return httpresponse.Error(w, httpresponse.ErrInternalError, err.Error())
	}

	// Response to client
	return response(w, types.PtrString(req.Format), result)
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

	// Cannot diarize when translating
	if req.Diarize != nil {
		return httpresponse.Error(w, httpresponse.ErrBadRequest, "Cannot diarize when translating")
	}

	// Start a translation task
	var result *schema.Transcription
	if err := service.WithModel(model, func(taskctx *task.Context) error {
		taskctx.SetTranslate(true)
		taskctx.SetDiarize(types.PtrBool(req.Diarize))

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
		result = taskctx.Result()

		// Decode, resample and segment the audio file
		return segment(ctx, w, taskctx, req.File.Body, func(seg *schema.Segment) {
			// TODO - for streaming
		})
	}); err != nil {
		return httpresponse.Error(w, httpresponse.ErrInternalError, err.Error())
	}

	// Response to client
	return response(w, types.PtrString(req.Format), result)
}

func segment(ctx context.Context, w http.ResponseWriter, taskctx *task.Context, r io.Reader, fn func(seg *schema.Segment)) error {
	// Create a segmenter
	segmenter, err := segmenter.NewReader(r, 0, whisper.SampleRate)
	if err != nil {
		return err
	}

	// Read segments and perform transcription or  translation
	if err := segmenter.DecodeFloat32(ctx, func(ts time.Duration, buf []float32) error {
		return taskctx.Transcribe(ctx, ts, buf, fn)
	}); err != nil {
		return err
	}

	// Return sucess
	return nil
}

const (
	FormatJson        = "json"
	FormatVerboseJson = "verbose_json"
	FormatText        = "text"
	FormatSrt         = "srt"
	FormatVtt         = "vtt"
)

func response(w http.ResponseWriter, format string, response *schema.Transcription) error {
	switch strings.ToLower(format) {
	case FormatJson, FormatVerboseJson:
		return httpresponse.JSON(w, http.StatusOK, 2, response)
	case FormatText, "":
		return httpresponse.Write(w, http.StatusOK, types.ContentTypeTextPlain, func(w io.Writer) (int, error) {
			return w.Write([]byte(response.Text))
		})
	case FormatSrt:
		return httpresponse.Write(w, http.StatusOK, "application/x-subrip", func(w io.Writer) (int, error) {
			for _, seg := range response.Segments {
				task.WriteSegmentSrt(w, seg)
			}
			return 0, nil
		})
	case FormatVtt:
		return httpresponse.Write(w, http.StatusOK, "text/vtt", func(w io.Writer) (int, error) {
			for _, seg := range response.Segments {
				task.WriteSegmentVtt(w, seg)
			}
			return 0, nil
		})
	}

	// Error - invalid format
	return httpresponse.ErrBadRequest.Withf("Invalid response format: %q", format)
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
