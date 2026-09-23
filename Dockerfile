# ==========================================
# Build the Go binary
# ==========================================
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

# Download Go modules first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source code
COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /build/aigateway ./main.go

# ==========================================
# Create a minimal runtime image
# ==========================================
FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder --chmod=755 /build/aigateway .
COPY --from=builder /build/casbin_model.conf ./casbin_model.conf

EXPOSE 8080

ENTRYPOINT ["/app/aigateway"]
