# Email Resource

A [Concourse](http://concourse.ci) resource that sends emails.

## Getting started
Look at the [demo pipeline](https://github.com/pivotal-cf/email-resource/blob/master/ci/demo-pipeline.yml).

This resource acts as an SMTP client, using `PLAIN` auth over TLS.  So you need an SMTP server that supports all that.

For development, we've been using [Amazon SES](https://aws.amazon.com/ses/) with its [SMTP support](http://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html)

## Source Configuration
All of the following configuration is required.
```
smtp:
  host: smtp.example.com
  port: "587" # this must be a string
  username: a-user
  password: my-password
from: build-system@example.com
to: [ "dev-team@example.com", "product@example.net" ]
```
Note that `to` is an array, and that `port` is a string.
If you're using `fly configure` with the `--vars-from` substitutions, every `{{ variable }}` 
[automatically gets converted to a string](http://concourse.ci/fly-cli.html).
But for literals you need to surround it with quotes.

## Behavior

This is an output-only resource, so `check` and `in` actions are no-ops.

### `out`: Send an email

#### Parameters

* `subject`: *Required.* Path to plain text file containing the subject
* `body`: *Required.* Path to file containing the email body.
* `send_empty_body`: *Optional.* If true, send the email even if the body is empty (defaults to `false`).


## Development
To install a development-version of the resource, you currently need to update your Concourse deployment manifest.

Under the `worker` job's `properties`, add this section:
```
      groundcrew:
        resource_types:
        - image: docker:///pcfseceng/email-resource
          type: email
        - image: /var/vcap/packages/archive_resource
          type: archive
        - image: /var/vcap/packages/cf_resource
          type: cf
        - image: /var/vcap/packages/docker_image_resource
          type: docker-image
        - image: /var/vcap/packages/git_resource
          type: git
        - image: /var/vcap/packages/s3_resource
          type: s3
        - image: /var/vcap/packages/semver_resource
          type: semver
        - image: /var/vcap/packages/time_resource
          type: time
        - image: /var/vcap/packages/tracker_resource
          type: tracker
        - image: /var/vcap/packages/pool_resource
          type: pool
        - image: /var/vcap/packages/vagrant_cloud_resource
          type: vagrant-cloud
        - image: /var/vcap/packages/github_release_resource
          type: github-release
        - image: /var/vcap/packages/bosh_io_release_resource
          type: bosh-io-release
        - image: /var/vcap/packages/bosh_io_stemcell_resource
          type: bosh-io-stemcell
        - image: /var/vcap/packages/bosh_deployment_resource
          type: bosh-deployment
```

Note that all but the first item are built-in (and may therefore be out-of-date).
