module github.com/DDP-Projekt/Kompilierer

go 1.22.2

require (
	github.com/bafto/Go-LLVM-Bindings v1.0.2
	github.com/google/go-github/v55 v55.0.0
	github.com/llir/irutil v0.0.0-20230226050352-c20f75c375f9
	github.com/llir/llvm v0.3.6
	github.com/minio/selfupdate v0.6.0
	github.com/otiai10/copy v1.14.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
)

replace github.com/bafto/Go-LLVM-Bindings => ../../Go-LLVM-Bindings/

require (
	aead.dev/minisign v0.3.0 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/cloudflare/circl v1.3.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mewmew/float v0.0.0-20211212214546-4fe539893335 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
)
