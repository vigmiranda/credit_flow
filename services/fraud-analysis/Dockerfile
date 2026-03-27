FROM golang:1.23 AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/fraud-analysis ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/fraud-analysis /app/fraud-analysis
EXPOSE 8086
ENTRYPOINT ["/app/fraud-analysis"]

