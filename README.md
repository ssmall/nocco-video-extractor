# NOCCO Video Extractor

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