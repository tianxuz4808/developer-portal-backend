# stage 1: build the go binary

# Tasks: this involves getting the golang docker image (use alpine as it has the smallest size)
# running the go build command to finish compiling the binary

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o developer-portal-backend

# stage 2: run the go binary
# Taks: This involves creating a new stage, retrieving the built go binary, 

FROM alpine:latest

COPY --from=builder /app/developer-portal-backend /developer-portal-backend

ENTRYPOINT [ "/developer-portal-backend" ]