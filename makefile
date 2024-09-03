# Format the code
fmt:
	go fmt ./...

# Vet the code
vet:
	go vet ./...

# Run the code
run:
	go run main.go

run-sidecar: fmt vet
	go run cmd/sidecar/main.go --config=./config.yaml
# Default target
all: fmt vet run