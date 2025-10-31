FROM golang:alpine

WORKDIR /

COPY go.mod go.sum ./

RUN go mod download

ENV PORT=3000

COPY . .

RUN go build -o main .

EXPOSE $PORT

CMD ["./main"]
