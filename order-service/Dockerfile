FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go env -w GOPROXY=https://proxy.golang.org,direct && go mod download

FROM gcr.io/distroless/base-debian12
COPY --from=build /bin/order-service /order-service
EXPOSE 8080
ENTRYPOINT ["/order-service"]
