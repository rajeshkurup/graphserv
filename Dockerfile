FROM golang:1.26-alpine AS build
RUN go install github.com/swaggo/swag/cmd/swag@latest
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init --parseDependency --parseInternal
RUN CGO_ENABLED=0 go build -o /graphserv .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /graphserv /graphserv
EXPOSE 8080
ENTRYPOINT ["/graphserv"]
