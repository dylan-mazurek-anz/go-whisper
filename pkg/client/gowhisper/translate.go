package gowhisper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	// Packages
	"github.com/mutablelogic/go-client"
	"github.com/mutablelogic/go-whisper/pkg/client/openai"
)

/////////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

// Translate performs a transcription request and returns the result in english
func (c *Client) Translate(ctx context.Context, req TranslationRequest) (*TranscriptionResponse, error) {
	var response TranscriptionResponse

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
	} else if err := c.Do(payload, &response, client.OptPath(openai.TranslatePath)); err != nil {
		return nil, err
	}

	// Return success
	return &response, nil
}
