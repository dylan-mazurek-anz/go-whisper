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
	"github.com/mutablelogic/go-whisper/pkg/client/gowhisper"
	"github.com/mutablelogic/go-whisper/pkg/client/openai"
	"github.com/mutablelogic/go-whisper/pkg/schema"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

type Client struct {
	openai     *openai.Client
	elevenlabs *elevenlabs.Client
	gowhisper  *gowhisper.Client
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

	// gowhisper client
	if endpoint := gowhisper_endpoint(); endpoint != "" {
		if client, err := gowhisper.New(endpoint, opts...); err != nil {
			return nil, err
		} else {
			self.gowhisper = client
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

func gowhisper_endpoint() string {
	return os.Getenv("WHISPER_URL")
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

// List models for transcription and translation
func (c *Client) ListModels(ctx context.Context) ([]schema.Model, error) {
	result := make([]schema.Model, 0, 10)
	if c.openai != nil {
		for _, model := range openai.Models {
			result = append(result, schema.Model{
				Id:   model,
				Path: "openai",
			})
		}
	}
	if c.elevenlabs != nil {
		for _, model := range elevenlabs.Models {
			result = append(result, schema.Model{
				Id:   model,
				Path: "elevenlabs",
			})
		}
	}
	if c.gowhisper != nil {
		models, err := c.gowhisper.ListModels(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, models...)
	}

	// Return success
	return result, nil
}

// Download model for use in transcription and translation
func (c *Client) DownloadModel(ctx context.Context, path string, fn func(cur, total uint64)) (*schema.Model, error) {
	switch {
	case c.gowhisper != nil:
		model, err := c.gowhisper.DownloadModel(ctx, path, fn)
		if err != nil {
			return nil, err
		}

		// Return success
		return model, nil
	}

	// Return error
	return nil, httpresponse.ErrNotImplemented.Withf("downloading is not supported")
}

// Delete existing model
func (c *Client) DeleteModel(ctx context.Context, model string) error {
	switch {
	case c.gowhisper != nil:
		return c.gowhisper.DeleteModel(ctx, model)
	}

	// Return error
	return httpresponse.ErrNotImplemented.Withf("delete is not supported")
}

// Transcribe performs a transcription request in the language of the speech
func (c *Client) Transcribe(ctx context.Context, model string, r io.Reader, opt ...Opt) (*schema.Transcription, error) {
	var response *schema.Transcription
	switch {
	case c.openai != nil && slices.Contains(openai.Models, model):
		if req, err := applyOpts(apiopenai, model, r, opt...); err != nil {
			return nil, err
		} else if resp, err := c.openai.Transcribe(ctx, req.openai); err != nil {
			return nil, err
		} else {
			response = resp.Segments()
		}
	case c.elevenlabs != nil && slices.Contains(elevenlabs.Models, model):
		if req, err := applyOpts(apielevenlabs, model, r, opt...); err != nil {
			return nil, err
		} else if resp, err := c.elevenlabs.Transcribe(ctx, req.elevenlabs); err != nil {
			return nil, err
		} else {
			response = resp.Segments()
		}
	case c.gowhisper != nil && model != "":
		if req, err := applyOpts(apigowhisper, model, r, opt...); err != nil {
			return nil, err
		} else if resp, err := c.gowhisper.Transcribe(ctx, req.gowhisper); err != nil {
			return nil, err
		} else {
			response = resp.Segments()
		}
	default:
		return nil, httpresponse.ErrNotImplemented.Withf("model %q is not supported", model)
	}

	// Return success
	return response, nil
}

// Translate performs a transcription request and returns the result in english
func (c *Client) Translate(ctx context.Context, model string, r io.Reader, opt ...Opt) (*schema.Transcription, error) {
	var response *schema.Transcription
	switch {
	case c.openai != nil && slices.Contains(openai.Models, model):
		if req, err := applyOpts(apiopenai, model, r, opt...); err != nil {
			return nil, err
		} else if resp, err := c.openai.Translate(ctx, req.openai.TranslationRequest); err != nil {
			return nil, err
		} else {
			response = resp.Segments()
		}
	case c.elevenlabs != nil && slices.Contains(elevenlabs.Models, model):
		return nil, httpresponse.ErrNotImplemented.Withf("translation with model %q is not supported", model)
	// TODO
	/*
		case c.gowhisper != nil && model != "":
			if req, err := applyOpts(apigowhisper, model, r, opt...); err != nil {
				return nil, err
			} else if resp, err := c.gowhisper.Translate(ctx, req.gowhisper.TranslationRequest); err != nil {
				return nil, err
			} else {
				response = resp.Segments()
			}
	*/
	default:
		return nil, httpresponse.ErrNotImplemented.Withf("model %q is not supported", model)
	}

	// Return success
	return response, nil
}

/*
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
