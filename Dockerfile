FROM golang:1.20

WORKDIR /app
ENV TZ="America/New_York"

COPY . .

RUN go mod tidy
RUN go build -o /now-api

EXPOSE 8080

CMD [ "/now-api" ]
