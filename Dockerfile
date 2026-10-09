# stage 1: build the go binary

# Tasks: this involves getting the golang docker image (use alpine as it has the smallest size)
# running the go build command to finish compiling the binary

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

RUN go install github.com/go-delve/delve/cmd/dlv@latest

RUN go build -gcflags="all=-N -l" -o developer-portal-backend

# stage 2: run the go binary
# Taks: This involves creating a new stage, retrieving the built go binary, 

FROM alpine:latest

COPY --from=builder /go/bin/dlv /usr/local/bin/dlv
COPY --from=builder /app/developer-portal-backend /developer-portal-backend
EXPOSE 40000
EXPOSE 8080

# CMD ["dlv", "exec", "/developer-portal-backend", "--headless", "--listen=:40000", "--api-version=2", "--accept-multiclient"]
CMD [ "/developer-portal-backend" ]