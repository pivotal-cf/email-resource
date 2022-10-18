# Email Resource

A [Concourse](http://concourse-ci.org) resource that sends emails.

## Getting started
Add the following [Resource Type](https://concourse-ci.org/resource-types.html) to your Concourse pipeline
```yaml
resource_types:
  - name: email
    type: docker-image
    source:
      repository: pcfseceng/email-resource
```

Look at the [demo pipeline](https://github.com/pivotal-cf/email-resource/blob/master/example/demo-pipeline.yml) for a complete example.

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
* `skip_ssl_validation`: *Optional.* Whether or not to skip ssl validation.  true/false are valid options.  If omitted default is false
* `ca_cert`: *Optional.* Certificates content to verify servers with custom certificates. Only considered if `skip_ssl_validation` is `false`.
* `host_origin`: *Optional.* Host to send `Hello` from.  If not provided `localhost` is used
* `login_auth`: *Optional.* This will enable the flag to use Login Auth for authenticated. true/false are valid options. If omitted default is false

Within source:
* `from`: *Required.* Email Address to be sent from.
* `to`: *Required.Conditionally.* Array of email addresses to send email to.  Not required if job params contains a file reference that has to recipients.
* `cc`: *Optional* Array of email addresses to cc send email to.
* `bcc`: *Optional* Array of email addresses to bcc send email to.

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

An exmaple using custom certificates:
```yaml
resources:
- name: send-an-email
  type: email
  source:
    smtp:
      host: smtp.example.com
      port: "587" # this must be a string
      anonymous: true
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
    from: build-system@example.com
    to: [ "dev-team@example.com", "product@example.net" ]
```
Note that `to` is an array, and that `port` is a string.
If you're using `fly configure` with the `--load-vars-from` (`-l`) substitutions, every `{{ variable }}`
[automatically gets converted to a string](http://concourse-ci.org/fly.html).
But for literals you need to surround it with quotes.

## Behavior

This is an output-only resource, so `check` and `in` actions are no-ops.

### `out`: Send an email

#### Parameters

* `headers`: *Optional.* Path to plain text file containing additional mail headers
* `subject`: *Optional.* Path to plain text file containing the subject. Either `subject` or `subject_text` required. `subject_text` takes precedence.
* `subject_text`: *Optional.* The subject as text. Either `subject` or `subject_text` required. `subject_text` takes precedence.
* `body`: *Optional.* Path to file containing the email body. Either `body` or `body_text` required. `body_text` takes precedence.
* `body_text`: *Optional.* The email body as text. Either `body` or `body_text` required. `body_text` takes precedence.
* `send_empty_body`: *Optional.* If true, send the email even if the body is empty (defaults to `false`).
* `to`: *Optional.* Path to plain text file containing recipients which could be determined at build time. You can run a task before, which figures out the email of the person who committed last to a git repository (`git -C $source_path --no-pager show $(git -C $source_path rev-parse HEAD) -s --format='%ae' > output/email.txt`).  This file can contain `,` delimited list of email address if wanting to send to multiples.
* `to_text`: *Optional.* The `,` delimited list of to addresses. `to_text` appends to any `to` in params or source
* `cc`: *Optional.* Path to plain text file containing recipients which could be determined at build time. This file can contain `,` delimited list of email address if wanting to send to multiples.
* `cc_text`: *Optional.* The `,` delimited list of cc addresses. `cc_text` appends to any `cc` in params or source
* `bcc`: *Optional.* Path to plain text file containing recipients which could be determined at build time. This file can contain `,` delimited list of email address if wanting to send to multiples.
* `bcc_text`: *Optional.* The `,` delimited list of bcc addresses. `bcc_text` appends to any `bcc` in params or source
* `debug`: *Optional.* If set to `true` additional information send to stderr
* `attachment_globs:` *Optional.* If provided will attach any file to the email that matches the glob path(s)

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

You can use the values below in any of the source files or text properties to access the corresponding metadata made available by concourse, as documented [here](https://concourse-ci.org/implementing-resource-types.html)

* `${BUILD_ID}`
* `${BUILD_NAME}`
* `${BUILD_JOB_NAME}`
* `${BUILD_PIPELINE_NAME}`
* `${ATC_EXTERNAL_URL}`
* `${BUILD_TEAM_NAME}`

For example:

```yaml
  - put: send-an-email
    params:
      subject_text: "Build finished: ${BUILD_PIPELINE_NAME}/${BUILD_JOB_NAME}/${BUILD_NAME}"
      body_text: "Build finished: ${ATC_EXTERNAL_URL}/teams/main/pipelines/${BUILD_PIPELINE_NAME}/jobs/${BUILD_JOB_NAME}/builds/${BUILD_NAME}"
```

#### HTML Email

To send HTML email set the `headers` parameter to a file containing the following:

```
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"
```


## Build from the source

`email-resource` is written in [Go](https://golang.org/).
To build the binary yourself, follow these steps:

* Install `Go`. (tested with 1.17)
* Clone the repo
  - `git clone git@github.com:pivotal-cf/email-resource.git`
  - `cd email-resource`
* Build or test:
  - `go test .\... -v`
  - `go build -o bin/check check/cmd/*.go`
  - `go build -o bin/in in/cmd/*.go`
  - `go build -o bin/out out/cmd/*.go`

To cross compile, set the `$GOOS` and `$GOARCH` environment variables.
For example: `GOOS=linux GOARCH=amd64 go build`.

## Testing

To run the unit tests, use `go test $(glide nv)`.
