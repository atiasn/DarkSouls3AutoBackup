build:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o DS3SaveBack.exe DarkSouls3AutoBackup.go
