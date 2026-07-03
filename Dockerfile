# Build a fully static, self-contained server image.
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/subscribe ./cmd/subscribe

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/subscribe /usr/local/bin/subscribe
EXPOSE 8080
ENV SUBSCRIBE_ADDR=:8080 SUBSCRIBE_NO_BROWSER=true
ENTRYPOINT ["subscribe", "serve"]
