# Use the official golang image as a base image for the backend build
FROM golang:1.19-buster AS backend-builder

# Set the working directory inside the backend-builder stage
WORKDIR /app

# Copy the backend code and build it
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./

# Build the backend executable without disabling CGO
RUN go build -o social-network-backend .

# Now, set up the production stage using Debian Buster as the base, which includes glibc
FROM debian:buster-slim

# Install ca-certificates for HTTPS and other dependencies your application may need
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory inside the production stage
WORKDIR /app

# Copy the built backend executable from the backend-builder stage
COPY --from=backend-builder /app/social-network-backend .

# Copy the pre-built frontend app (dist directory) to the appropriate location
COPY frontend/dist ./frontend/dist

# Copy the backend/database directory including the tables.sql to the working directory
COPY backend/datab ./datab

COPY backend/datab.db /app/

# Expose the port your application runs on
EXPOSE 8091

# Command to run the backend server
CMD ["./social-network-backend"]
