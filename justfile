all: run

build:
  go build

docker-build:
  docker build -t grombley -f grombley.dockerfile .

ci:
  just build
  just fmt

fmt:
  #!/usr/bin/env sh
  if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    gofmt -d -s -l .
    exit 1
  fi
  printf "\033[92mgofmt Success\033[0m\n"

fix-fmt:
  gofmt -w -s .

run:
  go run main.go

test:
  go test
  ./tests/runner.sh
