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
	httpresponse.JSON(w, respModels{
		Object: "list",
		Models: service.ListModels(),
	}, http.StatusOK, 2)
}

func DownloadModel(ctx context.Context, w http.ResponseWriter, r *http.Request, service *whisper.Whisper) {
	// Get query and body
	var query queryDownloadModel
	var req reqDownloadModel
	if err := httprequest.Query(&query, r.URL.Query()); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := httprequest.Body(&req, r); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate the request
	if err := req.Validate(); err != nil {
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
			httpresponse.Error(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	// Return the model information
	if query.Stream {
		stream.Write("ok", model)
	} else {
		httpresponse.JSON(w, model, http.StatusCreated, 2)
	}
}

func GetModelById(ctx context.Context, w http.ResponseWriter, service *whisper.Whisper, id string) {
	model := service.GetModelById(id)
	if model == nil {
		httpresponse.Error(w, http.StatusNotFound)
		return
	}
	httpresponse.JSON(w, model, http.StatusOK, 2)
}

func DeleteModelById(ctx context.Context, w http.ResponseWriter, service *whisper.Whisper, id string) {
	model := service.GetModelById(id)
	if model == nil {
		httpresponse.Error(w, http.StatusNotFound)
		return
	}
	if err := service.DeleteModelById(model.Id); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
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
