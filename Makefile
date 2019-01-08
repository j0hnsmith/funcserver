.PHONY: build lint


build:
	mkdir -p artifacts
	GOOS=linux CGO_ENABLED=0 go build -o artifacts/main cmd/example/main.go
	zip -j ./artifacts/main.zip ./artifacts/main

lint:
	gometalinter ./... --vendor --skip=vendor --exclude=\.*_mock\.*\.go --exclude=vendor\.* --cyclo-over=15 --deadline=10m --disable-all \
        --enable=errcheck \
        --enable=vet \
        --enable=deadcode \
        --enable=gocyclo \
        --enable=golint \
        --enable=varcheck \
        --enable=structcheck \
        --enable=vetshadow \
        --enable=ineffassign \
        --enable=interfacer \
        --enable=unconvert \
        --enable=goconst \
        --enable=gosimple \
        --enable=staticcheck \
        --enable=gosec