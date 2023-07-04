FROM golang:latest
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GO build -o /app/router
ENTRYPOINT /app/router
