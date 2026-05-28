FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/payment-service .

FROM alpine:3.23
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/payment-service .
EXPOSE 8080
ENTRYPOINT ["./payment-service"]
