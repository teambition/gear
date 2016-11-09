test:
	go test
	go test ./middleware

bench:
	go test -bench=.

cover:
	rm -f *.coverprofile
	go test -coverprofile=gear.coverprofile
	go test -coverprofile=middleware.coverprofile ./middleware
	gover
	go tool cover -html=gover.coverprofile

doc:
	godoc -http=:6060

.PHONY: test bench cover doc
