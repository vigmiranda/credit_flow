FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mock-oidc ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/mock-oidc /app/mock-oidc
EXPOSE 8090
ENTRYPOINT ["/app/mock-oidc"]
