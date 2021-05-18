FROM golang:1.16-alpine AS build
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV DOCKER_BUILDKIT=1
WORKDIR /src
COPY . .
RUN go install github.com/pingcap/failpoint/failpoint-ctl@latest \
    && /go/bin/failpoint-ctl enable \
    && go test ./... \
    && /go/bin/failpoint-ctl disable
RUN go build  -o /out/metamorphosis cmd/main.go
FROM alpine AS base
COPY --from=build /out/metamorphosis /
RUN addgroup -g 2000 metad && \
    adduser -H -D -s /bin/sh -u 2000 -G metad metad
USER metad
CMD /metamorphosis
