set GOPATH=%BuildFolder%
go get -v github.com/cookieo9/dropbox-go/dropbox
go test -v github.com/cookieo9/dropbox-go/dropbox
