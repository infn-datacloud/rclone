# rclone

Rclone patcher to add IAM support

## :rocket: Quick start

How to use rclone with IAM support

### :clipboard: Requirements

1. [oidc-agent](https://github.com/indigo-dc/oidc-agent) >= 4.0.x

   - with your oidc provider registered. To use **INFN-Cloud** for instance, follow the instructions [here](https://confluence.infn.it/pages/viewpage.action?spaceKey=INFNCLOUD&title=How+To%3A+Test+TOSCA+with+orchent) until reaching a working `oidc-token infncloud` command.

2. The rclone executable available in the [release page](https://github.com/DODAS-TS/rclone/releases)

### :pencil2: Configuration setup


You can use the `rclone config` command to create your configuration or create a config file with your favourite text editor.

<details>
<summary>1. rclone config</summary>

```bash
$ ./rclone config
Current remotes:

Name                 Type
====                 ====

e) Edit existing remote
n) New remote
d) Delete remote
r) Rename remote
c) Copy remote
s) Set configuration password
q) Quit config
e/n/d/r/c/s/q>n
```

type `n` and continue inserting the name to identify this configuration e.g. `infncloud`

```bash
name> infncloud
```

Then from the prompted llist of backends chose s3 (usually number 4)

```bash
Storage> 4
```

From the new list now chose `INFN Cloud S3 with STS IAM` (number 7 usually)

```bash
provider> 7
```

Then press enter at:

```bash
env_auth>
```

Now put in account the name of the oidc-agent profile you want to use. In other word the name that typing `oidc-token <your profile name>` is returning you with a valid token (in case of INFN-Cloud, would probably be `infncloud`).

```bash
account> <your profile name>
```

Then press enter for the next 3 questions:

```bash
AWS Access Key ID.
Leave blank for anonymous access or runtime credentials.
Enter a string value. Press Enter for the default ("").
access_key_id> 
AWS Secret Access Key (password)
Leave blank for anonymous access or runtime credentials.
Enter a string value. Press Enter for the default ("").
secret_access_key> 
Region to connect to.
Leave blank if you are using an S3 clone and you don't have a region.
Enter a string value. Press Enter for the default ("").
Choose a number from below, or type in your own value
 1 / Use this if unsure. Will use v4 signatures and an empty region.
   \ ""
 2 / Use this only if v4 signatures don't work, e.g. pre Jewel/v10 CEPH.
   \ "other-v2-signature"
region> 
```

Insert now, when prompted the minio endpoint (e.g. `https://minio.cloud.infn.it/`)

```bash
endpoint> https://minio.cloud.infn.it/
```

Now just press enter until the configuration ends and then type q to exit

```bash
Location constraint - must be set to match the Region.
Leave blank if not sure. Used when creating buckets only.
Enter a string value. Press Enter for the default ("").
location_constraint> 
Canned ACL used when creating buckets and storing or copying objects.

This ACL is used for creating objects and if bucket_acl isn't set, for creating buckets too.

For more info visit https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl

Note that this ACL is applied when server-side copying objects as S3
doesn't copy the ACL from the source but rather writes a fresh one.
Enter a string value. Press Enter for the default ("").
Choose a number from below, or type in your own value
 1 / Owner gets FULL_CONTROL. No one else has access rights (default).
   \ "private"
 2 / Owner gets FULL_CONTROL. The AllUsers group gets READ access.
   \ "public-read"
   / Owner gets FULL_CONTROL. The AllUsers group gets READ and WRITE access.
 3 | Granting this on a bucket is generally not recommended.
   \ "public-read-write"
 4 / Owner gets FULL_CONTROL. The AuthenticatedUsers group gets READ access.
   \ "authenticated-read"
   / Object owner gets FULL_CONTROL. Bucket owner gets READ access.
 5 | If you specify this canned ACL when creating a bucket, Amazon S3 ignores it.
   \ "bucket-owner-read"
   / Both the object owner and the bucket owner get FULL_CONTROL over the object.
 6 | If you specify this canned ACL when creating a bucket, Amazon S3 ignores it.
   \ "bucket-owner-full-control"
acl> 
Edit advanced config? (y/n)
y) Yes
n) No (default)
y/n> 
Remote config
--------------------
[infncloud]
type = s3
provider = INFN Cloud
account = cloud
endpoint = https://minio.cloud.infn.it/
--------------------
y) Yes this is OK (default)
e) Edit this remote
d) Delete this remote
y/e/d>
Current remotes:

Name                 Type
====                 ====
infncloud            s3

e) Edit existing remote
n) New remote
d) Delete remote
r) Rename remote
c) Copy remote
s) Set configuration password
q) Quit config
e/n/d/r/c/s/q> q
```

</details>

<details>
<summary>2. rclone config file from scratch</summary>

You can create a config file starting from this template:

```yaml
[infncloud]
type = s3
provider = INFN cloud
oidc_agent = true
account = username
env_auth = false
access_key_id =
secret_access_key =
session_token =
endpoint = https://minio.cloud.infn.it/
```

</details>

### :arrow_forward: Test it out

You can check the current configuration file used with the command `rclone config file`.

To use a specific config file you can use the argument `--config` and add the config file path.

If your configuration is working, you should be able to see your bucket content (also if and only if `oidc-token <your profile name>` is working in your session):

```bash
$ ./rclone --config configFile.conf ls infncloud:/<username>

# OUTPUT
Access token is: 3497bbf4bb72da964d196f...261c9e5d577ccfc9d9
     6145 fileA.py
104857600 fileB.img
[...]
```

## :wrench: Dev Tips

Here you will find some developer tips.

### :file_folder: Folders

* `backend/s3`: the new file that uses IAM and the base to create the patch
* `patches`: the patch generated comparing the original `s3` module with the new `s3iam` (the one in `backend/s3` folder)

### :hammer: How to dev and build

1. Updates the `RCLONE_VERSION` in the `Makefile` to match the latest stable release
2. `make clean`
3. Rename `s3.go` in `backend/s3` folder in `s3.go.old`
4. Copy the new `s3.go` file from `rclone/backend/s3`
5. Apply changes to `s3Connection` function in `backend/s3`
6. Verify with git diff the remain changes
7. `make build`
