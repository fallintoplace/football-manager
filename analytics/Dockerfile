FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/touchline-analytics .

FROM alpine:3.22

RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/touchline-analytics /touchline-analytics
EXPOSE 8787
ENTRYPOINT ["/touchline-analytics"]
