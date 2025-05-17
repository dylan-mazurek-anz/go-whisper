package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	// Packages
	"github.com/mutablelogic/go-server/pkg/httprequest"
	"github.com/mutablelogic/go-server/pkg/httpresponse"
	"github.com/mutablelogic/go-whisper"
	"github.com/mutablelogic/go-whisper/pkg/schema"
)

///////////////////////////////////////////////////////////////////////////////
// TYPES

type respModels struct {
	Object string          `json:"object,omitempty"`
	Models []*schema.Model `json:"models"`
}

type reqDownloadModel struct {
	Path string `json:"path"`
}

type queryDownloadModel struct {
	Stream bool `json:"stream"`
}

type respDownloadModelStatus struct {
	Status    string `json:"status"`
	Total     uint64 `json:"total,omitempty"`
	Completed uint64 `json:"completed,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func ListModels(ctx context.Context, w http.ResponseWriter, service *whisper.Whisper) {
	httpresponse.JSON(w, http.StatusOK, 2, respModels{
		Object: "list",
		Models: service.ListModels(),
	})
}

func DownloadModel(ctx context.Context, w http.ResponseWriter, r *http.Request, service *whisper.Whisper) {
	// Get query and body
	var query queryDownloadModel
	var req reqDownloadModel
	if err := httprequest.Query(r.URL.Query(), &query); err != nil {
		httpresponse.Error(w, httpresponse.ErrBadRequest, err.Error())
		return
	}
	if err := httprequest.Read(r, &req); err != nil {
		httpresponse.Error(w, httpresponse.ErrBadRequest, err.Error())
		return
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		httpresponse.Error(w, httpresponse.ErrBadRequest, err.Error())
		return
	}

	// Create a text stream
	var stream *httpresponse.TextStream
	if query.Stream {
		if stream = httpresponse.NewTextStream(w); stream == nil {
			httpresponse.Error(w, httpresponse.ErrInternalError, "Cannot create text stream")
			return
		}
		defer stream.Close()
	}

	// Download the model
	t := time.Now()
	model, err := service.DownloadModel(ctx, req.Name(), func(curBytes, totalBytes uint64) {
		if time.Since(t) > time.Second && stream != nil {
			t = time.Now()
			stream.Write("progress", respDownloadModelStatus{
				Status:    fmt.Sprint("downloading ", req.Name()),
				Total:     totalBytes,
				Completed: curBytes,
			})
		}
	})
	if err != nil {
		if stream != nil {
			stream.Write("error", err.Error())
		} else {
			httpresponse.Error(w, httpresponse.ErrGatewayError, err.Error())
		}
		return
	}

	// Return the model information
	if query.Stream {
		stream.Write("ok", model)
	} else {
		httpresponse.JSON(w, http.StatusCreated, 2, model)
	}
}

func GetModelById(ctx context.Context, w http.ResponseWriter, service *whisper.Whisper, id string) {
	model := service.GetModelById(id)
	if model == nil {
		httpresponse.Error(w, httpresponse.ErrNotFound, id)
		return
	}
	httpresponse.JSON(w, http.StatusOK, 2, model)
}

func DeleteModelById(ctx context.Context, w http.ResponseWriter, service *whisper.Whisper, id string) {
	model := service.GetModelById(id)
	if model == nil {
		httpresponse.Error(w, httpresponse.ErrNotFound, id)
		return
	}
	if err := service.DeleteModelById(model.Id); err != nil {
		httpresponse.Error(w, httpresponse.ErrInternalError, err.Error())
		return
	}
	httpresponse.Empty(w, http.StatusOK)
}

///////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

// Validate the request
func (r reqDownloadModel) Validate() error {
	if r.Path == "" {
		return errors.New("missing path")
	}
	return nil
}

// Return the model name
func (r reqDownloadModel) Name() string {
	return filepath.Base(r.Path)
}

// Return the model path
func (r reqDownloadModel) DestPath() string {
	return filepath.Dir(r.Path)
}
