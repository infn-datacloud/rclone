# rclone
Rclone patcher to add IAM support

## Quick start

### Pre-requisite

- oidc-agent >= 4.0.x
  with your oidc provider registered (e.g. INFN-Cloud IAM instance)

- download the rclone binary:

```bash
wget 
chmod +x ./rclone
```

### Configure s3 with STS and IAM manged by oidc-agent

Use `.rclone config` to setup your rclone 

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

From the new list now chose `INFN CLOUD S3 with STS IAM` (number 7 usually)

```bash
provider> 7
```

Then press enter at:

```bash
env_auth>
```

Now put in account the name of the oidc-agent profile you want to use. In other word the name that typing `oidc-token <your profile name>` is returning you with a valid token.

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
provider = INFN CLOUD
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

### Test it out

Now you should be able to see your bucket content (if and only if `oidc-token <your profile name>` is working in your session) with:

```bash
rclone ls infncloud:/
```

