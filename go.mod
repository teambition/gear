module github.com/teambition/gear

go 1.15

require (
	github.com/GitbookIO/mimedb v0.0.0-20180329142916-39fdfdb4def4 // indirect
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-http-utils/cookie v1.3.1
	github.com/go-http-utils/negotiator v1.0.0
	github.com/pelletier/go-toml v1.8.1
	github.com/soheilhy/cmux v0.1.4
	github.com/stretchr/testify v1.6.1
	github.com/teambition/compressible-go v1.0.1
	github.com/teambition/trie-mux v1.5.0
	github.com/vulcand/oxy v1.1.0
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/grpc v1.34.0
)

replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20181030102418-4d3f4d9ffa16
	golang.org/x/net => github.com/golang/net v0.0.0-20181102091132-c10e9556a7bc
	golang.org/x/sys => github.com/golang/sys v0.0.0-20181031143558-9b800f95dbbc
	golang.org/x/text => github.com/golang/text v0.3.1-0.20181010134911-4d1c5fb19474
	golang.org/x/tools => github.com/golang/tools v0.0.0-20181016205153-5ef16f43e633
	google.golang.org/genproto => github.com/google/go-genproto v0.0.0-20181016170114-94acd270e44e
	google.golang.org/grpc => github.com/grpc/grpc-go v1.26.0
)
