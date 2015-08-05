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
