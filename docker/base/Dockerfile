# ===========================================
#  Yao Base Image
#  A minimal base image for running Yao applications
# ===========================================
FROM debian:bullseye-slim

# Install necessary tools and clean up in one layer
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install Yao using the original script
RUN curl -fsSL https://raw.githubusercontent.com/sjzsdu/yao/refs/heads/main/get-yao.sh | bash

# Set working directory
WORKDIR /app
