ORIGINAL_FILE=rclone/backend/s3/s3.go
UPDATED_FILE=backend/s3/s3.go
PATCH_FILE=patches/s3.patch

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

patch: prepare
	patch ${ORIGINAL_FILE} < ${PATCH_FILE}
	cp backend/s3/iam.go rclone/backend/s3/