run:
	go run main.go

fmt:
	go fmt ./...

test:
	go mod download
	mkdir testresults || true
	go test -coverprofile=testresults/testcoverage.txt  -race ./...
	go tool cover -html=testresults/testcoverage.txt -o testresults/testcoverage.html

# Run end-to-end tests against a live Nudgebee environment. Requires
# NUDGEBEE_INTEGRATION=1, NUDGEBEE_ENDPOINT, and NUDGEBEE_API_KEY to be set;
# otherwise individual tests will t.Skip themselves.
test-e2e:
	go mod download
	NUDGEBEE_INTEGRATION=1 go test -tags=e2e -race ./...

benchmark:
	go test -bench=. -count=5 -benchmem -run=^$  ./... | tee testresults/testperf.txt

lint:
	golangci-lint run

validate: lint test

BIN ?= nbctl

build: validate
	go build -o dist/$(BIN) .

build-artifact:
	go build -o dist/$(BIN) .

install: build
	mkdir -p ~/go/bin
	cp dist/nbctl ~/go/bin/nbctl
	codesign -s - -f ~/go/bin/nbctl
