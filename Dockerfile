# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

# The bazil.org/fuse package requires CGO (C compiler) and FUSE development headers.
RUN apk update && apk add --no-cache gcc musl-dev fuse3-dev

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the executable. CGO_ENABLED=1 is required for FUSE.
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o /usr/local/bin/biggieLossFs ./cmd/fuse-server/


# Stage 2: Create a minimal final image
FROM alpine:latest
RUN mkdir /mnt/biggie

RUN apk update && apk add --no-cache fuse3

# Copy the executable from the builder stage
COPY --from=builder /usr/local/bin/biggieLossFs /usr/local/bin/biggieLossFs

# Set the entrypoint for reference.
ENTRYPOINT ["/usr/local/bin/biggieLossFs"]
