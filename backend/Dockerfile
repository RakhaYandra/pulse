FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pulse ./cmd/pulse

FROM alpine:3.20
WORKDIR /app
COPY --from=build /pulse ./pulse
EXPOSE 8080
ENTRYPOINT ["./pulse"]
