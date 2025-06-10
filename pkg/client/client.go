package client

import (
	"context"
	"io"
	"os"
	"slices"

	// Packages
	"github.com/mutablelogic/go-client"
	"github.com/mutablelogic/go-server/pkg/httpresponse"
	"github.com/mutablelogic/go-whisper/pkg/client/elevenlabs"
	"github.com/mutablelogic/go-whisper/pkg/client/openai"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

type Client struct {
	openai     *openai.Client
	elevenlabs *elevenlabs.Client
}

///////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

// New creates a new client, with openai, elevenlabs and other clients
func New(opts ...client.ClientOpt) (*Client, error) {
	self := new(Client)

	// openai client
	if key := openai_key(); key != "" {
		if client, err := openai.New(key, opts...); err != nil {
			return nil, err
		} else {
			self.openai = client
		}
	}

	// elevenlabs client
	if key := elevenlabs_key(); key != "" {
		if client, err := elevenlabs.New(key, opts...); err != nil {
			return nil, err
		} else {
			self.elevenlabs = client
		}
	}

	// Return success
	return self, nil
}

///////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func openai_key() string {
	return os.Getenv("OPENAI_API_KEY")
}

func elevenlabs_key() string {
	return os.Getenv("ELEVENLABS_API_KEY")
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

// List models for transcription and translation
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	result := make([]string, 0, 10)
	if c.openai != nil {
		result = append(result, openai.Models...)
	}
	if c.elevenlabs != nil {
		result = append(result, elevenlabs.Models...)
	}
	// Return success
	return result, nil
}

// Transcribe performs a transcription request in the language of the speech
func (c *Client) Transcribe(ctx context.Context, model string, r io.Reader, opt ...Opt) error {
	switch {
	case c.openai != nil && slices.Contains(openai.Models, model):
		if req, err := applyOpts(model, r, opt...); err != nil {
			return err
		} else if _, err := c.openai.Transcribe(ctx, req.TranscriptionRequest); err != nil {
			return err
		}
	case c.elevenlabs != nil && slices.Contains(elevenlabs.Models, model):
		if req, err := applyOpts(model, r, opt...); err != nil {
			return err
		} else if _, err := c.elevenlabs.Transcribe(ctx, req.TranscribeRequest); err != nil {
			return err
		}
	default:
		return httpresponse.ErrNotImplemented.Withf("model %q is not supported", model)
	}

	// Return success
	return nil
}

// Translate performs a transcription request and returns the result in english
func (c *Client) Translate(ctx context.Context, model string, r io.Reader, opt ...Opt) error {
	switch {
	case c.openai != nil && slices.Contains(openai.Models, model):
		if req, err := applyOpts(model, r, opt...); err != nil {
			return err
		} else if _, err := c.openai.Translate(ctx, req.TranscriptionRequest.TranslationRequest); err != nil {
			return err
		}
	case c.elevenlabs != nil && slices.Contains(elevenlabs.Models, model):
		return httpresponse.ErrNotImplemented.Withf("translation with model %q is not supported", model)
	default:
		return httpresponse.ErrNotImplemented.Withf("model %q is not supported", model)
	}

	// Return success
	return nil
}

/*
///////////////////////////////////////////////////////////////////////////////
// PING

func (c *Client) Ping(ctx context.Context) error {
	return c.DoWithContext(ctx, client.MethodGet, nil, client.OptPath("health"))
}

///////////////////////////////////////////////////////////////////////////////
// MODELS

func (c *Client) ListModels(ctx context.Context) ([]schema.Model, error) {
	var models struct {
		Models []schema.Model `json:"models"`
	}
	if err := c.DoWithContext(ctx, client.MethodGet, &models, client.OptPath("models")); err != nil {
		return nil, err
	}
	// Return success
	return models.Models, nil
}

func (c *Client) DeleteModel(ctx context.Context, model string) error {
	return c.DoWithContext(ctx, client.MethodDelete, nil, client.OptPath("models", model))
}

func (c *Client) DownloadModel(ctx context.Context, path string, fn func(status string, cur, total int64)) (schema.Model, error) {
	var req struct {
		Path string `json:"path"`
	}
	type resp struct {
		schema.Model
		Status    string `json:"status"`
		Total     int64  `json:"total,omitempty"`
		Completed int64  `json:"completed,omitempty"`
	}

	// stream=true for progress reports
	query := url.Values{}
	if fn != nil {
		query.Set("stream", "true")
	}

	// Download the model
	req.Path = path

	var r resp
	if payload, err := client.NewJSONRequest(req); err != nil {
		return schema.Model{}, err
	} else if err := c.DoWithContext(ctx, payload, &r,
		client.OptPath("models"),
		client.OptQuery(query),
		client.OptNoTimeout(),
		client.OptTextStreamCallback(func(evt client.TextStreamEvent) error {
			switch evt.Event {
			case "progress":
				var r resp
				if err := evt.Json(&r); err != nil {
					return err
				} else {
					fn(r.Status, r.Completed, r.Total)
				}
			case "error":
				var errstr string
				if evt.Event == "error" {
					if err := evt.Json(&errstr); err != nil {
						return err
					} else {
						return errors.New(errstr)
					}
				}
			case "ok":
				if err := evt.Json(&r); err != nil {
					return err
				}
			}
			return nil
		}),
	); err != nil {
		return schema.Model{}, err
	}

	// Return success
	return r.Model, nil
}

func (c *Client) Transcribe(ctx context.Context, model string, r io.Reader, opt ...Opt) (*schema.Transcription, error) {
	var request struct {
		File  multipart.File `json:"file"`
		Model string         `json:"model"`
		opts
	}
	var response schema.Transcription

	// Get the name from the io.Reader
	name := ""
	if f, ok := r.(*os.File); ok {
		name = filepath.Base(f.Name())
	} else {
		name = "audio.wav" // Default name if not a file
	}

	// Create the request
	request.Model = model
	request.File = multipart.File{
		Path: name,
		Body: r,
	}
	for _, o := range opt {
		if err := o(&request.opts); err != nil {
			return nil, err
		}
	}

	// Request->Response
	if payload, err := client.NewMultipartRequest(request, types.ContentTypeFormData); err != nil {
		return nil, err
	} else if err := c.DoWithContext(ctx, payload, &response, client.OptPath("audio/transcriptions"), client.OptNoTimeout()); err != nil {
		return nil, err
	}

	// Return success
	return &response, nil
}

func (c *Client) Translate(ctx context.Context, model string, r io.Reader, opt ...Opt) (*schema.Transcription, error) {
	var request struct {
		File  multipart.File `json:"file"`
		Model string         `json:"model"`
		opts
	}
	var response schema.Transcription

	// Get the name from the io.Reader
	name := ""
	if f, ok := r.(*os.File); ok {
		name = filepath.Base(f.Name())
	} else {
		name = "audio.wav" // Default name if not a file
	}

	// Create the request
	request.Model = model
	request.File = multipart.File{
		Path: name,
		Body: r,
	}
	for _, o := range opt {
		if err := o(&request.opts); err != nil {
			return nil, err
		}
	}

	// Request->Response
	if payload, err := client.NewMultipartRequest(request, types.ContentTypeFormData); err != nil {
		return nil, err
	} else if err := c.DoWithContext(ctx, payload, &response, client.OptPath("audio/translations"), client.OptNoTimeout()); err != nil {
		return nil, err
	}

	// Return success
	return &response, nil
}
*/
