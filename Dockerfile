FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o image-server

EXPOSE 3000
CMD ["./image-server"]