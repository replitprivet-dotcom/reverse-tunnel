FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/tunnel-server ./cmd/tunnel-server && \
    go build -trimpath -ldflags='-s -w' -o /out/tunnel-client ./cmd/tunnel-client

FROM alpine:3.20
RUN addgroup -S tunnel && adduser -S -G tunnel tunnel
COPY --from=build /out/tunnel-server /usr/local/bin/tunnel-server
COPY --from=build /out/tunnel-client /usr/local/bin/tunnel-client
USER tunnel
ENTRYPOINT ["/usr/local/bin/tunnel-server"]
