module httpcli

go 1.18

require (
	github.com/peterbourgon/ff/v3 v3.1.2
	golang.org/x/net v0.0.0-20220403103023-749bd193bc2b
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	tracer v0.0.0-00010101000000-000000000000
)

replace tracer => ../tracer

require golang.org/x/text v0.3.7 // indirect
