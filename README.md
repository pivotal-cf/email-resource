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

### `source`:

#### Parameters

Within smtp:

* `host`: *Required.* SMTP Host name
* `port`: *Required.* SMTP Port, must be entered as a string
* `anonymous`: *Optional.* Whether or not to require credential.  true/false are valid options.  If omitted default is false
* `username`: *Required, Conditionally.* Username to authenticate with.  Ignored if `anonymous: true`
* `password`: *Required, Conditionally.* Password to authenticate with.  Ignored if `anonymous: true`

Within source:
* `from`: *Required.* Email Address to be sent from.
* `to`: *Required.Conditionally.* Array of email addresses to send email to.  Not required if job params contains a file reference that has to recipients.

An example source configuration is below.
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

An example source configuration is below supporting sending email when anonymous is permitted.
```yaml
resources:
- name: send-an-email
  type: email
  source:
    smtp:
      host: smtp.example.com
      port: "587" # this must be a string
      anonymous: true
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
* `body`: *Required.* Path to file containing the email body.
* `send_empty_body`: *Optional.* If true, send the email even if the body is empty (defaults to `false`).
* `to`: *Optional.* Path to plain text file containing recipients which could be determined at build time. You can run a task before, which figures out the email of the person who committed last to a git repository (`git -C $source_path --no-pager show $(git -C $source_path rev-parse HEAD) -s --format='%ae' > output/email.txt`).  This file can contain `,` delimited list of email address if wanting to send to multiples.

For example, a build plan might contain this:
```yaml
  - put: send-an-email
    params:
      subject: generated-subject-file
      body: generated-body-file
```

For example, a build plan might contain this if using generated list of recipient(s):
```yaml
  - put: send-an-email
    params:
      subject: generated-subject-file
      body: generated-body-file
      to: generated-to-file
```

You can use the values below in any of the source files to access the corresponding metadata made available by concourse, as documented [here](http://concourse.ci/implementing-resources.html)

* `${BUILD_ID}`
* `${BUILD_NAME}`
* `${BUILD_JOB_NAME}`
* `${BUILD_PIPELINE_NAME}`
* `${ATC_EXTERNAL_URL}`

For example `generated-subject` could have content `Build ${BUILD_JOB_NAME} failed` which would result in the subject sent to be `Build job-name failed`

#### HTML Email

To send HTML email set the `headers` parameter to a file containing the following:

```
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"
```
