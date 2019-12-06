############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git && apk  --no-cache add ca-certificates

# Create appuser.
RUN adduser -D -g '' appuser
WORKDIR $GOPATH/src/wolfmqttbridge


COPY go.mod go.sum ./
RUN go mod download


# Assuming the source code is collocated to this Dockerfile
COPY . .
# Fetch dependencies.

# Using go get.
#RUN go get -d -v

# Build the binary.
#RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/wolfmqttbridge
RUN  CGO_ENABLED=0  GOOS=linux  GOARCH=amd64 go build  -ldflags='-w -extldflags "-static" -s' -o /wolfmqttbridge

############################
# STEP 2 build a small image
############################
FROM scratch
# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
# Copy our static executable.
COPY --from=builder /wolfmqttbridge /wolfmqttbridge

#COPY certs roots required to validate outbound HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Use an unprivileged user.
USER appuser
# Run the binary.
ENTRYPOINT ["/wolfmqttbridge"]