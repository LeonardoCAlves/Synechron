FROM golang:1.27.3-alpine AS build

WORKDIR /src/order-platform
COPY order-platform/go.mod ./
RUN go mod download
COPY order-platform/ ./
ARG SERVICE
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/service /service
EXPOSE 8081 8082
ENTRYPOINT ["/service"]
