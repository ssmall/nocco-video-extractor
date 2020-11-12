// Package http contains functionality related to serving http requests
package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ssmall/nocco-video-extractor/pkg/drive"
	"github.com/ssmall/nocco-video-extractor/pkg/video"
)

// ClipExtractionHandler creates a http.HandlerFunc that handles requests to
// extract video clips from Google Drive files and reupload them to Drive.
func ClipExtractionHandler(d drive.Client, e video.Extractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body ExtractionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		start, err := parseDuration(body.ClipStartTime)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		end, err := parseDuration(body.ClipEndTime)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		filename, contents, err := d.GetFile(r.Context(), body.SourceFileID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer contents.Close()

		f, err := os.Create(path.Join(os.TempDir(), filename))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		defer f.Close()
		defer os.Remove(f.Name())

		_, err = io.Copy(f, contents)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		transcode, err := e.Clip(r.Context(), f.Name(), start, end)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		defer transcode.Close()

		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		newFilename := fmt.Sprintf("%s_%s_to_%s%s", base, body.ClipStartTime, body.ClipEndTime, ext)

		log.Printf("Uploading clip as %q", newFilename)

		url, err := d.UploadFile(r.Context(), newFilename, body.DestinationFolderID, transcode)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		resp, err := json.Marshal(&ExtractionResponse{FileURL: url})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(resp)
	}
}

func parseDuration(timestamp string) (time.Duration, error) {
	r := regexp.MustCompile(`^(\d{2}):(\d{2}):(\d{2})$`)
	matches := r.FindStringSubmatch(timestamp)
	if matches == nil {
		return 0, fmt.Errorf("%q does not match format HH:MM:SS", timestamp)
	}
	return time.ParseDuration(fmt.Sprintf("%sh%sm%ss", matches[1], matches[2], matches[3]))
}
