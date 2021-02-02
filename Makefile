ORIGINAL_FILE=rclone/backend/s3/s3.go
UPDATED_FILE=backend/s3/s3.go
PATCH_FILE=patches/s3.patch

CUR_OS=""

ifeq ($(OS),Windows_NT)
publish:
	cp `go env GOPATH`/bin/rclone`go env GOEXE` ./rclone_windows.exe
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
endif

.PHONY: all build prepare clean patch vars version

all: clean prepare patch build

build: clean prepare patch
	# Change dir and call `make rclone`
	$(MAKE) -C rclone rclone

clean:
	git submodule init
	git submodule update --init
	git submodule sync
	cd rclone && git reset --hard HEAD

prepare: clean
	-diff -u ${ORIGINAL_FILE} ${UPDATED_FILE} > ${PATCH_FILE}
	patch ${ORIGINAL_FILE} < ${PATCH_FILE}

patch: prepare
	cp backend/s3/iam.go rclone/backend/s3/
	cd rclone && go mod tidy