FROM golang:1.23 AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/credit-analysis ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/credit-analysis /app/credit-analysis
EXPOSE 8085
ENTRYPOINT ["/app/credit-analysis"]

