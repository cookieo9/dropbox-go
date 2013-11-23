set GOPATH=%BuildFolder%
go get -v github.com/cookieo9/dropbox-go
go test -v github.com/cookieo9/dropbox-go
