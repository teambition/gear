test:
	go test --race
	go test --race ./logging
	go test --race ./middleware/cors
	go test --race ./middleware/favicon
	go test --race ./middleware/grpc
	go test --race ./middleware/secure
	go test --race ./middleware/static
	go test --race ./middleware/requestid

bench:
	go test -bench=.

cover:
	rm -f *.coverprofile
	go test -coverprofile=gear.coverprofile
	go test -coverprofile=logging.coverprofile ./logging
	go test -coverprofile=cors.coverprofile ./middleware/cors
	go test -coverprofile=favicon.coverprofile ./middleware/favicon
	go test -coverprofile=grpc.coverprofile ./middleware/grpc
	go test -coverprofile=static.coverprofile ./middleware/static
	go test -coverprofile=secure.coverprofile ./middleware/secure
	go test -coverprofile=requestid.coverprofile ./middleware/requestid
	gover
	go tool cover -html=gover.coverprofile
	rm -f *.coverprofile

doc:
	godoc -http=:6060

.PHONY: test bench cover doc
