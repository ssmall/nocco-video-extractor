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

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ssmall/nocco-video-extractor/pkg/drive"
	noccohttp "github.com/ssmall/nocco-video-extractor/pkg/http"
	"github.com/ssmall/nocco-video-extractor/pkg/video"
)

const defaultPort = 8080
const terminationWait = 10 * time.Second

func main() {
	var driveUser string
	if s, ok := os.LookupEnv("DRIVE_USER"); ok {
		driveUser = s
	} else {
		log.Fatalln("Google Drive user must be specified via environment variable DRIVE_USER")
	}
	log.Printf("Impersonating user %q", driveUser)

	var port int
	if s, ok := os.LookupEnv("PORT"); ok {
		p, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("Expected integer value for PORT but got %q: %v", s, err)
		}
		port = p
	} else {
		port = defaultPort
	}

	ctx := context.Background()

	d, err := drive.NewClient(ctx, driveUser)

	if err != nil {
		log.Fatalln("Error initializing Google Drive client:", err)
	}

	r := mux.NewRouter()
	r.Handle("/extract", noccohttp.ClipExtractionHandler(d, video.NewExtractor()))

	log.Println("Starting server on port", port)
	srv := &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), terminationWait)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Shutting down")
	os.Exit(0)
}
