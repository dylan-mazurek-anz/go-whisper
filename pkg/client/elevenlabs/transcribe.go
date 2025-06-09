package elevenlabs

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

func (c *Client) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	var response TranscribeResponse

	// Set default model
	if req.Model == "" {
		req.Model = ElevenLabsModels[0]
	} else if !slices.Contains(ElevenLabsModels, req.Model) {
		return nil, fmt.Errorf("invalid model %q, must be one of %v", req.Model, ElevenLabsModels)
	}

	// Check file
	if req.File.Body == nil {
		return nil, fmt.Errorf("file is required")
	}
	if req.File.Path == "" {
		if f, ok := req.File.Body.(*os.File); ok {
			req.File.Path = filepath.Base(f.Name())
		}
	}

	if payload, err := client.NewMultipartRequest(req, client.ContentTypeAny); err != nil {
		return nil, err
	} else if err := c.Do(payload, &response, client.OptPath(ElevenLabsTranscribe)); err != nil {
		return nil, err
	}

	// Return success
	return &response, nil
}
