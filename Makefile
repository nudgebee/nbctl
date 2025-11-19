run:
	go run main.go

fmt:
	go fmt ./...

test:
	go mod download
	mkdir testresults || true
	go test -coverprofile=testresults/testcoverage.txt  -race ./...
	go tool cover -html=testresults/testcoverage.txt -o testresults/testcoverage.html

benchmark:
	go test -bench=. -count=5 -benchmem -run=^$  ./... | tee testresults/testperf.txt

lint:
	golangci-lint run

validate: lint test

build: validate
	go build -o dist/nbctl .

install: build
	mkdir -p ~/go/bin
	cp dist/nbctl ~/go/bin/nbctl
	codesign -s - -f ~/go/bin/nbctl
