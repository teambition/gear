test:
	go test
	go test ./logging
	go test ./middleware/favicon
	go test ./middleware/static

bench:
	go test -bench=.

cover:
	rm -f *.coverprofile
	go test -coverprofile=gear.coverprofile
	go test -coverprofile=logging.coverprofile ./logging
	go test -coverprofile=favicon.coverprofile ./middleware/favicon
	go test -coverprofile=static.coverprofile ./middleware/static
	gover
	go tool cover -html=gover.coverprofile

doc:
	godoc -http=:6060

.PHONY: test bench cover doc
