# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Modules first so this layer is cached until go.mod/go.sum change — a
# source-only edit then does not re-download every dependency.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary: no CGO, no libc dependency in the runtime image.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime stage ----
FROM alpine:3.21

# ca-certificates is required — Resend and SMTP both speak TLS to the
# outside world, and Alpine ships without a trust store.
RUN apk add --no-cache ca-certificates

# Hugging Face runs containers as UID 1000; match it so nothing needs
# root at runtime.
RUN adduser -D -u 1000 app

WORKDIR /app

COPY --from=build /out/api /app/api
# Migrations travel with the image so the migration step can run them from
# the same place the schema they describe was written.
COPY --chown=app:app migrations /app/migrations

USER app

# Matches app_port in the Space README. config.Load defaults PORT to 8080
# when the variable is unset, so this holds whether or not the platform
# injects one.
ENV PORT=8080
EXPOSE 8080

CMD ["/app/api"]
