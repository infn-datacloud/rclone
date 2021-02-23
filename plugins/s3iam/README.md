## CREATE AN RCLONE PLUGIN

Implementing the instructions [here](https://github.com/rclone/rclone/blob/5b84adf3b983f6208341ebcbbc3f8b3fbfccdb97/CONTRIBUTING.md#writing-a-plugin)

This makes possible to produce the .so needed to load a custom plugin. The IAM implementation is slighltly different w.r.t. sts-wire version: the access token is generated from oidc-agent running on the host through this library: [https://indigo-dc.gitbook.io/oidc-agent/api/api-go](https://indigo-dc.gitbook.io/oidc-agent/api/api-go)


Example:

```bash
curl https://rclone.org/install.sh | sudo bash
make build-plugin
export RCLONE_PLUGIN_PATH=$PWD/plugins/s3iam/
rclone config
```

You should now see s3iam as an option.