FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rx-trace ./

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/rx-trace /usr/local/bin/rx-trace
EXPOSE 8080
ENTRYPOINT ["rx-trace"]
