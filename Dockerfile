FROM golang:1.20
WORKDIR /usr/src/app/url_shortener
COPY cmd ./cmd
COPY config ./config
COPY internal ./internal
COPY db-data ./db-data

COPY go.mod ./
COPY go.sum ./
COPY my.env ./
COPY docker-compose.yml ./

RUN go mod download && go mod verify
COPY . .

RUN go build cmd/url_shortener/main.go

CMD ["./main"]

# RUN cd cmd/url_shortener
# CMD ["go", "build", "./url_shortener"]
# CMD ["go", "build", "./cmd/url_shortener/url_shortener"]

