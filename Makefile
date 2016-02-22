
# Build binary for Linux
arpingall: arpingall.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build arpingall.go

clean:
	$(RM) arpingall
