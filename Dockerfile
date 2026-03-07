FROM golang:1.25-alpine AS builder

WORKDIR /app

# copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# run unit test
RUN set -o pipefail && go test ./... | grep -v "no test files"

# Compile a static binary
# CGO_ENABLED=0 removes C dependencies for perfect portability
# -ldflags="-s -w" strips debug symbols to drastically reduce file size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

#final stage
# use an absolutelty empty image
FROM scratch

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]