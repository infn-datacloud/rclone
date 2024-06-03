ORIGINAL_FILE=rclone/backend/s3/s3.go
UPDATED_FILE=backend/s3/s3.go
PATCH_FILE=patches/s3.patch
RCLONE_VERSION=v1.58.1

CUR_OS=""

ifeq ($(OS),Windows_NT)
publish:
	cp `go env GOPATH`/bin/rclone`go env GOEXE` ./rclone_windows.exe
prepare: clean
	copy backend\s3\s3.go rclone\backend\s3\s3.go
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
publish:
	cp `go env GOPATH`/bin/rclone`go env GOEXE` ./rclone_linux
    endif
    ifeq ($(UNAME_S),Darwin)
publish:
	cp `go env GOPATH`/bin/rclone`go env GOEXE` ./rclone_osx
    endif
prepare: clean
	-diff -u ${ORIGINAL_FILE} ${UPDATED_FILE} > ${PATCH_FILE}
	patch ${ORIGINAL_FILE} < ${PATCH_FILE}
endif

.PHONY: all build-macos build prepare clean patch vars version build-plugin patch-windows build-windows

all: clean prepare patch build build-macos patch-windows build-windows

build-windows: clean prepare patch-windows
	.\win-build.bat
	$(MAKE) GOTAGS=cmount -C rclone rclone

build-macos: clean prepare patch
	# Change dir and call `make rclone`
	$(MAKE) GOTAGS=cmount -C rclone rclone

build: clean prepare patch
	# Change dir and call `make rclone`
	$(MAKE) -C rclone rclone

clean:
	git submodule init
	git submodule update --init
	git submodule sync
	cd rclone && git reset --hard HEAD && git checkout ${RCLONE_VERSION}

patch-windows: prepare
	copy backend\s3\iam.go rclone\backend\s3\iam.go
	copy backend\s3\authMinio.go rclone\backend\s3\authMinio.go
	cd rclone && go mod tidy

patch: prepare
	cp backend/s3/iam.go rclone/backend/s3/
	cp backend/s3/authMinio.go rclone/backend/s3/
	cd rclone && go mod tidy

build-plugin:
	cd plugins/s3iam && go mod tidy && go build -buildmode=plugin -o librcloneplugin_backend_s3iam.so  .
