.PHONY: test
test:
	go test ./... -count=1

.PHONY: test-cover
test-cover:
	go test ./... -coverprofile unit_test_coverage.out -count=1 && go tool cover -func=unit_test_coverage.out

.PHONY: bench
bench:
	go test -run none -bench . -benchtime 3s -benchmem -memprofile memprofile.out

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy
