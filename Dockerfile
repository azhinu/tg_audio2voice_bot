FROM golang:1.24-alpine AS builder

ARG VERSION=dev
ARG REPO_URL=https://github.com/azhinu/audio2voice

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags "-X 'main.version=${VERSION}' -X 'main.repoURL=${REPO_URL}'" \
	-o /bin/audio2voice ./main.go

FROM alpine:3.20

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app
COPY --from=builder /bin/audio2voice /usr/local/bin/audio2voice

ENV TG_A2V_PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/audio2voice"]
