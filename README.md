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

This resource acts as an SMTP client, using `PLAIN` auth over TLS.  So you need an SMTP server that supports all that.

For development, we've been using [Amazon SES](https://aws.amazon.com/ses/) with its [SMTP support](http://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html)

## Source Configuration
An example source configuration is below.  None of the parameters are optional.
```yaml
resources:
- name: send-an-email
  type: email
  source:
    smtp:
      host: smtp.example.com
      port: "587" # this must be a string
      username: a-user
      password: my-password
    from: build-system@example.com
    to: [ "dev-team@example.com", "product@example.net" ] #optional if `params.additional_recipient` is specified
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
* `additional_recipient`: *Optional.* Path to plain text file containing additional recipient which could be determined at build time. You can run a task before, which figures out the email of the person who committed last to a git repository (`git -C $source_path --no-pager show $(git -C $source_path rev-parse HEAD) -s --format='%ae' > output/email.txt`)
* `subject`: *Required.* Path to plain text file containing the subject
* `body`: *Required.* Path to file containing the email body.
* `send_empty_body`: *Optional.* If true, send the email even if the body is empty (defaults to `false`).

For example, a build plan might contain this:
```yaml
  - put: send-an-email
    params:
      subject: demo-prep-sha-email/generated-subject
      body: demo-prep-sha-email/generated-body
```

You can use the values below in any of the source files to access the corresponding metadata made available by concourse, as documented [here](http://concourse.ci/implementing-resources.html)

* `${BUILD_ID}`
* `${BUILD_NAME}`
* `${BUILD_JOB_NAME}`
* `${BUILD_PIPELINE_NAME}`
* `${ATC_EXTERNAL_URL}`

For example `demo-prep-sha-email/generated-subject` could have content `Build ${BUILD_JOB_NAME} failed` which would result in the subject sent to be `Build demo-email-an-artifact-sha failed`

#### HTML Email

To send HTML email set the `headers` parameter to a file containing the following:

```
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"
```
