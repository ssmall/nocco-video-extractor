# NOCCO Video Extractor

[![GitHub Super-Linter](https://github.com/ssmall/nocco-video-extractor/workflows/Lint%20Code%20Base/badge.svg)](https://github.com/marketplace/actions/super-linter)
[![codecov](https://codecov.io/gh/ssmall/nocco-video-extractor/branch/main/graph/badge.svg?token=19JLB7JO0I)](undefined)

This is a Golang microservice used to extract segments from video files
and upload the results to Google Drive.

It is purpose-built for use by the [North Corner Chamber Orchestra][].

## Building

This repository comes with a [Dockerfile][] that can be used to build
the project.

```bash
docker build -t nocco-video-extractor:latest .
```

[North Corner Chamber Orchestra]: https://nocco.org
[Dockerfile]: Dockerfile
