# --- Stage 1: runtime only ---
FROM gcr.io/distroless/base-debian12

# Set the working directory inside the container
WORKDIR /app

# Copy the pre-built Go binary into the image
COPY javanese-chess .

# Expose the API port
EXPOSE 9000

# Run the binary directly (no shell, no OS bloat)
ENTRYPOINT ["/app/javanese-chess"]
