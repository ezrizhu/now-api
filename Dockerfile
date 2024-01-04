FROM golang:1.20

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o /now-api

EXPOSE 8080

CMD [ "/now-api" ]
