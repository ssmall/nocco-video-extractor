# Copyright 2020 Spencer Small
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.15 as builder

WORKDIR /src

COPY . .

RUN go build -v -o /build/app ./cmd/main.go

FROM debian:buster-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ffmpeg=7:4.1.6-1~deb10u1 \
      openssl=1.1.1d-0+deb10u3 \
      ca-certificates=20200601~deb10u1 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /build/app .

ENTRYPOINT [ "./app" ]