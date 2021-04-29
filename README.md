# rclone

Rclone patcher to add IAM support

## :file_folder: Folders

* `backend/s3`: the new file that uses IAM and the base to create the patch
* `patches`: the patch generated comparing the original `s3` module with the new `s3iam` (the one in `backend/s3` folder)
