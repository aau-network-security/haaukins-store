FROM golang:1.13.0
MAINTAINER "Gian Marco Mennecozzi"
WORKDIR /haaukins

COPY . .
RUN go build -o server .
EXPOSE 50051
CMD ["./server"]
