FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG APP
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${APP}

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/app /app
USER app

ENTRYPOINT ["/app"]

