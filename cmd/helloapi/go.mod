module github.com/labiraus/go-utils/pkg/cmd/helloapi

go 1.23.3

require (
	github.com/labiraus/go-utils/pkg/api v0.0.0
	github.com/labiraus/go-utils/pkg/base v0.0.0
	github.com/labiraus/go-utils/pkg/prometheusutil v0.0.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/labiraus/go-utils/pkg/api => ../../pkg/api

replace github.com/labiraus/go-utils/pkg/base => ../../pkg/base

replace github.com/labiraus/go-utils/pkg/prometheusutil => ../../pkg/prometheusutil