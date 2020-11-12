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

// +build integration

package drive

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/drive/v3"
)

func uploadTestFile(t *testing.T, user string) (id, name, contents string) {
	t.Helper()

	name = t.Name()
	contents = "this is an integration test file for " + t.Name()

	ctx := context.Background()
	srv, err := getDriveService(ctx, user)

	if err != nil {
		t.Fatal(err)
	}

	f, err := srv.Files.Create(&drive.File{Name: name}).Media(bytes.NewBufferString(contents)).Context(ctx).Do()

	if err != nil {
		t.Fatal(err)
	}

	id = f.Id
	t.Cleanup(func() {
		srv.Files.Delete(id).Do()
	})

	return
}

func createTestFolder(t *testing.T, user string) string {
	t.Helper()

	ctx := context.Background()
	srv, err := getDriveService(ctx, user)

	if err != nil {
		t.Fatal(err)
	}

	folder, err := srv.Files.Create(&drive.File{
		Name:     t.Name(),
		MimeType: "application/vnd.google-apps.folder",
	}).Context(ctx).Do()

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		srv.Files.Delete(folder.Id).Do()
	})

	return folder.Id
}

func fileIDFromURL(t *testing.T, url string) string {
	r := regexp.MustCompile(`^https://drive.google.com/file/d/([a-zA-Z0-9_]+).*$`)

	matches := r.FindStringSubmatch(url)

	if matches == nil {
		t.Fatalf("url %q did not match expected format", url)
	}

	return matches[1]
}

// This test has the following external dependencies:
// - the GOOGLE_APPLICATION_CREDENTIALS environment variable must be set and must reference credentials that can be used for read/write on Google Drive
// - the TEST_USER environment variable must be set to the email address of a valid user in the GSuite organization
func TestGetFile(t *testing.T) {
	user, ok := os.LookupEnv("TEST_USER")
	if !ok {

		t.Fatal("Expected environment variable TEST_USER to be set")
	}
	id, name, contents := uploadTestFile(t, user)

	ctx := context.Background()

	c, err := NewClient(ctx, user)

	if err != nil {
		t.Fatal(err)
	}

	actualName, r, err := c.GetFile(ctx, id)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Close()

	actualContents, err := ioutil.ReadAll(r)

	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(name, actualName); diff != "" {
		t.Error("Filename different than expected (+got -want):", diff)
	}

	if diff := cmp.Diff(contents, string(actualContents)); diff != "" {
		t.Error("File contents different than expected (+got -want):", diff)
	}
}

// This test has the following external dependencies:
// - the GOOGLE_APPLICATION_CREDENTIALS environment variable must be set and must reference credentials that can be used for read/write on Google Drive
// - the TEST_USER environment variable must be set to the email address of a valid user in the GSuite organization
func TestUploadFile(t *testing.T) {
	user, ok := os.LookupEnv("TEST_USER")
	if !ok {

		t.Fatal("Expected environment variable TEST_USER to be set")
	}

	folderID := createTestFolder(t, user)

	ctx := context.Background()

	c, err := NewClient(ctx, user)

	if err != nil {
		t.Fatal(err)
	}

	expectedContents := "test file contents for " + t.Name()
	expectedName := t.Name()

	url, err := c.UploadFile(ctx, expectedName, folderID, bytes.NewBufferString(expectedContents))

	if err != nil {
		t.Fatal(err)
	}

	fileID := fileIDFromURL(t, url)

	name, r, err := c.GetFile(ctx, fileID)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Close()

	contents, err := ioutil.ReadAll(r)

	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedName, name); diff != "" {
		t.Error("Filename different than expected (+got -want):", diff)
	}

	if diff := cmp.Diff(expectedContents, string(contents)); diff != "" {
		t.Error("File contents different than expected (+got -want):", diff)
	}
}
