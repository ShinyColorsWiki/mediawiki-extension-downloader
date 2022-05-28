FROM golang:latest as builder
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go get -d -v ./...
RUN  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ./main ./...

FROM gcr.io/distroless/base
COPY --from=builder /app/main .
EXPOSE 3000
ENTRYPOINT ["./main"]