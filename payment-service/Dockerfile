FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go env -w GOPROXY=https://proxy.golang.org,direct && go mod download

FROM gcr.io/distroless/base-debian12
COPY --from=build /bin/payment-service /payment-service
EXPOSE 8081
ENTRYPOINT ["/payment-service"]
