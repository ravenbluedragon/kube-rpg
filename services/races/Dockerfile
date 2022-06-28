FROM golang:1.18-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o races

FROM alpine
WORKDIR /app
COPY --from=build /app/races /app/races
EXPOSE 8080

CMD ["/app/races"]