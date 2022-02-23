FROM golang:alpine as builder

RUN apk update && apk upgrade && \
    apk add --no-cache git

# Set necessary environmet variables needed for our image
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .

# Copy the code into the container
COPY . .

# Build the application
# go build -o [name] [path to file]
RUN go build -o app server.go

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/app .

############################
# STEP 2 build a small image
############################
FROM alpine:latest
RUN apk --no-cache add ca-certificates

COPY . .
COPY --from=builder /dist/app /
#COPY ./database/data.json /database/data.json
# Copy the code into the container
RUN mkdir -p db/data
# Command to run the executable
ENTRYPOINT ["/app"]