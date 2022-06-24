module github.com/dodas-ts/rclone/plugins/s3iam

go 1.15

require (
	github.com/aws/aws-sdk-go v1.35.20
	github.com/indigo-dc/liboidcagent-go v0.3.0
	github.com/minio/minio v0.0.0-20210223002953-2a79ea033206
	github.com/ncw/swift v1.0.52
	github.com/pkg/errors v0.9.1
	github.com/rclone/rclone v1.54.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace (
	github.com/aws/aws-sdk-go v0.0.0 => github.com/aws/aws-sdk-go v1.35.20
	github.com/ncw/swift v0.0.0 => github.com/ncw/swift v1.0.52
)
