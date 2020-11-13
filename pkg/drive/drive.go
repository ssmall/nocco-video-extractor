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

// Package drive provides methods for interacting with Google Drive
package drive

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Client provides an interface to fetch files from Google Drive
type Client interface {
	// GetFile gets the file with the given id.
	// Return values are: the name of the file, the contents of the file, and any errors that occurred.
	GetFile(ctx context.Context, id string) (string, io.ReadCloser, error)

	// UploadFile uploads a file with the given name and contents to the specified folder.
	// Returns the URL of the uploaded file.
	UploadFile(ctx context.Context, name, folder string, contents io.Reader) (string, error)
}

type driveClient struct {
	srv *drive.Service
}

func getClient(ctx context.Context) (*http.Client, error) {
	client, err := google.DefaultClient(ctx, drive.DriveScope)

	if err != nil {
		return nil, err
	}
	return client, nil
}

func getDriveService(ctx context.Context) (*drive.Service, error) {
	c, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(c))
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// NewClient creates a new DriveClient
func NewClient(ctx context.Context) (Client, error) {
	srv, err := getDriveService(ctx)
	if err != nil {
		return nil, err
	}
	return &driveClient{srv}, nil
}

func (c *driveClient) GetFile(ctx context.Context, id string) (string, io.ReadCloser, error) {
	f, err := c.srv.Files.Get(id).SupportsAllDrives(true).Context(ctx).Fields(googleapi.Field("name")).Do()
	if err != nil {
		return "", nil, fmt.Errorf("error getting filename: %w", err)
	}
	log.Printf("File %s has name %q", id, f.Name)
	r, err := c.srv.Files.Get(id).SupportsAllDrives(true).Context(ctx).Download()
	if err != nil {
		return "", nil, err
	}
	log.Printf("<--- %s %s, ContentLength: %d bytes", r.Status, r.Request.URL, r.ContentLength)
	return f.Name, r.Body, nil
}

func (c *driveClient) UploadFile(ctx context.Context, name, folder string, contents io.Reader) (string, error) {
	f, err := c.srv.Files.Create(&drive.File{
		Name:    name,
		Parents: []string{folder},
	}).SupportsAllDrives(true).Context(ctx).Media(contents).Fields("name", "id", "webViewLink").Do()
	if err != nil {
		return "", err
	}
	log.Printf("File uploaded as %q (id: %s) to folder %q", f.Name, f.Id, folder)
	return f.WebViewLink, nil
}
