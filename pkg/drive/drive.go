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
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Client provides an interface to fetch files from Google Drive
type Client interface {
	// GetFileContents gets the contents of the file with the given id
	GetFileContents(ctx context.Context, id string) (io.ReadCloser, error)
}

type driveClient struct {
	srv *drive.Service
}

func getClient(ctx context.Context, user string) (*http.Client, error) {
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)

	if err != nil {
		return nil, err
	}

	config, err := google.JWTConfigFromJSON(creds.JSON, drive.DriveScope)
	if err != nil {
		return nil, err
	}

	config.Subject = user

	return config.Client(ctx), nil
}

// NewClient creates a new DriveClient that impersonates the given user
func NewClient(ctx context.Context, user string) (Client, error) {
	c, err := getClient(ctx, user)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(c))
	if err != nil {
		return nil, err
	}
	return &driveClient{srv}, nil
}

func (c *driveClient) GetFileContents(ctx context.Context, id string) (io.ReadCloser, error) {
	r, err := c.srv.Files.Get(id).Context(ctx).Download()
	if err != nil {
		return nil, err
	}
	log.Printf("<--- %s %s, ContentLength: %d bytes", r.Status, r.Request.URL, r.ContentLength)
	return r.Body, nil
}
