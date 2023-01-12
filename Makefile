test:
	go test -v -tags=test --race ./...

bench:
	go test -bench=.

cover:
	go test -v -failfast -tags=test -timeout="3m" -coverprofile="./coverage.out" -covermode="atomic" ./...

doc:
	godoc -http=:6060

.PHONY: test bench cover doc
