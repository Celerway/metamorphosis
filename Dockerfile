FROM golang:1.18-alpine AS build
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV DOCKER_BUILDKIT=1
WORKDIR /src
COPY . .
RUN apk add git mosquitto mosquitto-clients
RUN /src/get-toxiproxy.sh
RUN go test ./... && \
    go build  -o /out/metamorphosis cmd/main.go
FROM alpine AS base
COPY --from=build /out/metamorphosis /
RUN addgroup -g 2000 metad && \
    adduser -H -D -s /bin/sh -u 2000 -G metad metad
USER metad
CMD /metamorphosis
