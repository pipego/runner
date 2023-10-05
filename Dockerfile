FROM golang:latest AS build-stage
WORKDIR /go/src/app
COPY . .
RUN make build

FROM gcr.io/distroless/base-debian11 AS production-stage
WORKDIR /
COPY --from=build-stage /go/src/app/bin/runner /
USER nonroot:nonroot
EXPOSE 29090
CMD ["/runner", "--listen-url=:29090"]
