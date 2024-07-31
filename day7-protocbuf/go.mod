module main

go 1.19

require (
	gocache v0.0.0
	google.golang.org/protobuf v1.34.2
)

require github.com/golang/protobuf v1.5.0 // indirect

replace gocache => ./gocache
