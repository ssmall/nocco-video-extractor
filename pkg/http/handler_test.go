// Copyright 2020 Spencer Small
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type closingBuffer struct {
	b *bytes.Buffer
}

func (b *closingBuffer) Close() error {
	return nil
}

func (b *closingBuffer) Read(p []byte) (n int, err error) {
	return b.b.Read(p)
}

type fakeDriveClient struct {
	// Stub outputs
	filename       string
	fileContents   closingBuffer
	createdFileURL string

	// Stub errors
	getFileError error
	uploadError  error

	// Capture inputs
	getFileID          string
	uploadFileName     string
	uploadFileFolder   string
	uploadFileContents []byte
}

func (c *fakeDriveClient) GetFile(ctx context.Context, id string) (string, io.ReadCloser, error) {
	if c.getFileError != nil {
		return "", nil, c.getFileError
	}
	c.getFileID = id
	return c.filename, &c.fileContents, nil
}

func (c *fakeDriveClient) UploadFile(ctx context.Context, name, folder string, contents io.Reader) (string, error) {
	if c.uploadError != nil {
		return "", c.uploadError
	}
	var err error
	c.uploadFileName = name
	c.uploadFileFolder = folder
	c.uploadFileContents, err = ioutil.ReadAll(contents)
	return c.createdFileURL, err
}

type fakeExtractor struct {
	// Stub outputs
	contents closingBuffer

	// Stub errors
	err error

	// Capture inputs
	clipFilename string
	clipStart    time.Duration
	clipEnd      time.Duration
}

func (e *fakeExtractor) Clip(ctx context.Context, filename string, start time.Duration, end time.Duration) (io.ReadCloser, error) {
	if e.err != nil {
		return nil, e.err
	}
	e.clipFilename = filename
	e.clipStart = start
	e.clipEnd = end
	return &e.contents, nil
}

func createRequest(t *testing.T, requestJSON string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/extract", bytes.NewBuffer([]byte(requestJSON)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("content-type", "application/json")
	return req
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		input    string
		expected time.Duration
	}{
		{
			input:    "12:34:56",
			expected: 12*time.Hour + 34*time.Minute + 56*time.Second,
		},
		{
			input:    "00:00:00",
			expected: 0,
		},
		{
			input:    "00:00:90",
			expected: 90 * time.Second,
		},
	}
	for _, test := range cases {
		t.Run(test.input, func(t *testing.T) {
			actual, err := parseDuration(test.input)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.expected, actual); diff != "" {
				t.Error("parseDuration output different than expected (+got -want):", diff)
			}
		})
	}
}

func TestParseDuration_error(t *testing.T) {
	_, err := parseDuration("asdasd")

	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestHandler_HappyPath(t *testing.T) {
	drive := &fakeDriveClient{
		filename:       "originalFile.fileExt",
		fileContents:   closingBuffer{bytes.NewBufferString("original file contents")},
		createdFileURL: "https://example.com",
	}
	extractor := &fakeExtractor{
		contents: closingBuffer{bytes.NewBufferString("clip contents")},
	}
	handler := ClipExtractionHandler(drive, extractor)

	requestJSON := `{
		"sourceFileId": "sourceFileId",
		"clipStartTime": "00:01:23",
		"clipEndTime": "00:02:34",
		"destinationFolderId": "destinationFolderId"
		}`

	req := createRequest(t, requestJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if drive.getFileID != "sourceFileId" {
		t.Errorf("got GetFile(context, %q), want GetFile(context, %q) ", drive.getFileID, "sourceFileId")
	}

	clipExt := filepath.Ext(extractor.clipFilename)
	expectedStart := 1*time.Minute + 23*time.Second
	expectedEnd := 2*time.Minute + 34*time.Second
	if clipExt != filepath.Ext(drive.filename) || extractor.clipStart != expectedStart || extractor.clipEnd != expectedEnd {
		t.Errorf("got Clip(context, <filename>%s, %s, %s), want Clip(context, <filename>%s, %s, %s)", clipExt, extractor.clipStart, extractor.clipEnd, filepath.Ext(drive.filename), expectedStart, expectedEnd)
	}

	expectedUploadName := "originalFile_00:01:23_to_00:02:34.fileExt"
	if drive.uploadFileName != expectedUploadName || drive.uploadFileFolder != "destinationFolderId" || cmp.Diff(drive.uploadFileContents, []byte("clip contents")) != "" {
		t.Errorf("got UploadFile(context, %q, %q, %q), want UploadFile(context, %q, %q, %q)", drive.uploadFileName, drive.uploadFileFolder, drive.uploadFileContents, expectedUploadName, "destinationFolderId", []byte("clip contents"))
	}

	if diff := cmp.Diff(http.StatusCreated, rr.Code); diff != "" {
		t.Fatal("Different response code than expected (+got -want):", diff)
	}

	expected := ExtractionResponse{
		FileURL: drive.createdFileURL,
	}

	var actual ExtractionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &actual); err != nil {
		t.Fatalf("Invalid response %q: %v", rr.Body, err)
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("Different response than expected (+got -want):", diff)
	}
}

func TestHandler_Error(t *testing.T) {
	validRequest := `{
		"sourceFileId": "sourceFileId",
		"clipStartTime": "00:01:23",
		"clipEndTime": "00:02:34",
		"destinationFolderId": "destinationFolderId"
		}`

	cases := []struct {
		name                 string
		requestBody          string
		drive                *fakeDriveClient
		extractor            *fakeExtractor
		expectedResponseCode int
	}{
		{
			name:                 "Invalid Request (not a JSON object)",
			requestBody:          "blah",
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			name:                 "Invalid Request (invalid start time)",
			requestBody:          `{"clipStartTime": "blah", "clipEndTime": "00:02:34"}`,
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			name:                 "Invalid Request (invalid end time)",
			requestBody:          `{"clipStartTime": "00:02:34", "clipEndTime": "blah"}`,
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			name:        "Error getting file from drive",
			requestBody: validRequest,
			drive: &fakeDriveClient{
				getFileError: errors.New("expected error"),
			},
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name:        "Transcoding error",
			requestBody: validRequest,
			drive: &fakeDriveClient{
				filename:     "test file",
				fileContents: closingBuffer{bytes.NewBufferString("file contents don't matter")},
			},
			extractor: &fakeExtractor{
				err: errors.New("expected error"),
			},
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name:        "Upload error",
			requestBody: validRequest,
			drive: &fakeDriveClient{
				filename:     "test file",
				fileContents: closingBuffer{bytes.NewBufferString("file contents don't matter")},
				uploadError:  errors.New("expected error"),
			},
			extractor: &fakeExtractor{
				contents: closingBuffer{bytes.NewBufferString("transcode contents")},
			},
			expectedResponseCode: http.StatusInternalServerError,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			handler := ClipExtractionHandler(test.drive, test.extractor)

			req := createRequest(t, test.requestBody)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if diff := cmp.Diff(test.expectedResponseCode, rr.Code); diff != "" {
				t.Fatal("Different response code than expected (+got -want):", diff)
			}
		})
	}
}
