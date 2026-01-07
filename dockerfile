# build bin container
FROM golang:1.24.11 AS build

WORKDIR /myBot

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bot cmd/bot/main.go

# build container
FROM alpine:latest

COPY --from=build /myBot/bot ./
COPY --from=build /myBot/internal/common/config/prod.yaml ./prod.yaml
COPY --from=build /myBot/migrations/ ./migrations/
COPY --from=build /myBot/dictionary.json ./dictionary.json

ENTRYPOINT ["/bot"]