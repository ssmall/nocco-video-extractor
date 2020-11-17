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
	"flag"
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

var readTimeout = flag.Duration("readtimeout", 15*time.Second, "Sets the read timeout for incoming HTTP requests")
var writeTimeout = flag.Duration("writetimeout", 15*time.Second, "Sets the write timeout for HTTP responses")
var idleTimeout = flag.Duration("idletimeout", 60*time.Second, "Sets the idle timeout for HTTP keepalive")

func main() {
	flag.Parse()
	log.Println("Read timeout is:", readTimeout)
	log.Println("Write timeout is:", writeTimeout)
	log.Println("Idle timeout is:", idleTimeout)

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

	d, err := drive.NewClient(ctx)

	if err != nil {
		log.Fatalln("Error initializing Google Drive client:", err)
	}

	r := mux.NewRouter()
	r.Handle("/extract", noccohttp.ClipExtractionHandler(d, video.NewExtractor()))

	log.Println("Starting server on port", port)
	srv := &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(port),
		WriteTimeout: *writeTimeout,
		ReadTimeout:  *readTimeout,
		IdleTimeout:  *idleTimeout,
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
