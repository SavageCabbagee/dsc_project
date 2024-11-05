FROM golang:1.23.2

WORKDIR /app

COPY . .

# Build the application
RUN go build -o raft-server

# Command will be provided via docker-compose
CMD ["./raft-server"]