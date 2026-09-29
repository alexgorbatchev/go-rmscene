set dotenv-load := false

default:
    @just --list

# Run unit tests
test:
    go test -v ./...

# Run checks
check: test
