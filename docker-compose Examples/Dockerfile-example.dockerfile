# -------- build stage -------------
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -ldflags="-s -w" -o /cred-wrapper

# -------- runtime stage -----------
FROM alpine:3.20
RUN adduser -D -h /nonexistent credwrap
USER credwrap
COPY --from=build /cred-wrapper /cred-wrapper
ENTRYPOINT ["/cred-wrapper"]