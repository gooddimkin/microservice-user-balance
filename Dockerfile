FROM golang:alpine as builder
RUN apk update && apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
# RUN apk --no-cache add ca-certifacates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["./main"]