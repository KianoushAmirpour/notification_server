FROM golang:1.24.5 AS build
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o myapp ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /prod

COPY --from=build /app/myapp /prod/myapp

EXPOSE 8000

ENTRYPOINT [ "/prod/myapp" ]