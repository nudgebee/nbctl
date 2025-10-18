FROM golang:1.21-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/nbctl .

FROM alpine:3.18

WORKDIR /app

COPY --from=build-stage /app/nbctl /app/nbctl

ENTRYPOINT ["/app/nbctl"]
