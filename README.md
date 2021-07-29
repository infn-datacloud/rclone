# rclone

Rclone patcher to add IAM support

## :file_folder: Folders

* `backend/s3`: the new file that uses IAM and the base to create the patch
* `patches`: the patch generated comparing the original `s3` module with the new `s3iam` (the one in `backend/s3` folder)

## :hammer: How to dev and build

1. Updates the `RCLONE_VERSION` in the `Makefile` to match the latest stable release
2. `make clean`
3. Rename `s3.go` in `backend/s3` folder in `s3.go.old`
4. Copy the new `s3.go` file from `rclone/backend/s3`
5. Apply changes to `s3Connection` function in `backend/s3`
6. Verify with git diff the remain changes
7. `make build`
