FROM golang:1.26-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed

FROM alpine:3.23

WORKDIR /app
RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S app -G app
COPY --from=build --chown=app:app /out/api /app/api
COPY --from=build --chown=app:app /out/migrate /app/migrate
COPY --from=build --chown=app:app /out/seed /app/seed
COPY --chown=app:app migrations /app/migrations
USER app
EXPOSE 8080
CMD ["/app/api"]
