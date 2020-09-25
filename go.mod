module github.com/teambition/gear

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/GitbookIO/mimedb v0.0.0-20180329142916-39fdfdb4def4 // indirect
	github.com/codahale/hdrhistogram v0.9.0 // indirect
	github.com/dimfeld/httptreemux v5.0.1+incompatible // indirect
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-http-utils/cookie v1.3.1
	github.com/go-http-utils/negotiator v1.0.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mailgun/timetools v0.0.0-20170619190023-f3a7b8ffff47 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pelletier/go-toml v1.8.1
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/teambition/compressible-go v1.0.1
	github.com/teambition/trie-mux v1.4.2
	github.com/vulcand/oxy v1.1.0
	golang.org/x/net v0.0.0-20200923182212-328152dc79b1
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/grpc v1.32.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
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

go 1.14
