# Email Resource

A [Concourse](http://concourse.ci) resource that sends emails.

## Getting started
Add the following [Resource Type](http://concourse.ci/configuring-resource-types.html) to your Concourse pipeline
```yaml
resource_types:
  - name: email
    type: docker-image
    source:
      repository: pcfseceng/email-resource
```

Look at the [demo pipeline](https://github.com/pivotal-cf/email-resource/blob/master/ci/demo-pipeline.yml) for a complete example.

This resource acts as an SMTP client.

For development, we've been using [Amazon SES](https://aws.amazon.com/ses/) with its [SMTP support](http://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html)

## Source Configuration

The `username` and `password` are optional. If they are omitted, the email is send without AUTH and does not use TLS.
It is using `PLAIN` auth over TLS otherwise. In that case you need an SMTP server that supports all that.

#### Parameters

```yaml
   smtp:
     host: <host>
     port: <port> # this must be a string
     username: <optional username>
     password: <optional password>
   from: <from address>
   to: [ <recipient addresses> ]
```

An example source configuration is below.
```yaml
resources:
- name: send-an-email
  type: email
  source:
    smtp:
      host: smtp.example.com
      port: "587" # this must be a string
      username: a-user # TLS and PLAIN AUTH enabled
      password: my-password
    from: build-system@example.com
    to: [ "dev-team@example.com", "product@example.net" ]
```
Note that `to` is an array, and that `port` is a string.
If you're using `fly configure` with the `--load-vars-from` (`-l`) substitutions, every `{{ variable }}` 
[automatically gets converted to a string](http://concourse.ci/fly-cli.html).
But for literals you need to surround it with quotes.

## Behavior

This is an output-only resource, so `check` and `in` actions are no-ops.

### `out`: Send an email

#### Parameters

* `headers`: *Optional.* Path to plain text file containing additional mail headers
* `subject`: *Required.* Path to plain text file containing the subject

One of the following has to be provided. If both `body` and `body_file` are provided, `body` takes precedence:
* `body`: Body as plain text
* `body_file`: Path to file containing the email body
* `send_empty_body`: If true, send the email even if the body is empty (defaults to `false`).

All body parameters support the [concourse build metadata parameters](http://concourse.ci/implementing-resources.html#resource-metadata).
*Important:* Only parameter expansion with braces is supported, e.g. `${BUILD_NAME}`. Only the parameters listed on the concourse page are supported.

For example, a build plan might contain this:
```yaml
  - put: send-an-email
    params:
      subject: demo-prep-sha-email/generated-subject
      body: "Link: ${ATC_EXTERNAL_URL}/pipelines/${BUILD_PIPELINE_NAME}/jobs/${BUILD_JOB_NAME}/builds/${BUILD_NAME}"
```

#### HTML Email

To send HTML email set the `headers` parameter to a file containing the following:

```
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"
```
