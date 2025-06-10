package openai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	// Packages
	"github.com/mutablelogic/go-client"
)

/////////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

// Transcribe performs a transcription request in the language of the speech
func (c *Client) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error) {
	var response TranscriptionResponse

	// Set default model
	if req.Model == "" {
		req.Model = Models[0]
	} else if !slices.Contains(Models, req.Model) {
		return nil, fmt.Errorf("invalid model %q, must be one of %v", req.Model, Models)
	}

	// Check file, set path if not provided
	if req.File.Body == nil {
		return nil, fmt.Errorf("file is required")
	} else if req.File.Path == "" {
		if f, ok := req.File.Body.(*os.File); ok {
			req.File.Path = filepath.Base(f.Name())
		}
	}

	// Create multipart request, and execute it
	if payload, err := client.NewMultipartRequest(req, client.ContentTypeAny); err != nil {
		return nil, err
	} else if err := c.Do(payload, &response, client.OptPath(TranscribePath)); err != nil {
		return nil, err
	}

	// Return success
	return &response, nil
}
