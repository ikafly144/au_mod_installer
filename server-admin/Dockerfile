# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git and nodejs/npm for tailwindcss
RUN apk add --no-cache git nodejs npm

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Setup tailwindcss build context
WORKDIR /app/server-admin
COPY server-admin/package.json ./
RUN npm install

# Copy only necessary files for css build
COPY server-admin/tailwind.config.js server-admin/input.css ./
COPY server-admin/templates/views ./templates/views

# Build css and backup
RUN mkdir -p templates/static && npm run build:css && cp templates/static/output.css /tmp/output.css

# Copy rest of the source code
WORKDIR /app
COPY . .

# Restore css
RUN cp /tmp/output.css server-admin/templates/static/output.css

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o server-admin ./server-admin

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/server-admin .

# Expose port
EXPOSE 8081

# Run the application
ENTRYPOINT ["./server-admin"]
CMD ["-addr", ":8081", "-valkey", "valkey:6379"]
