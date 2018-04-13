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

Look at the [demo pipeline](https://github.com/pivotal-cf/email-resource/blob/master/example/demo-pipeline.yml) for a complete example.

This resource allows retrieval of emails via the IMAP protocol. Only TLS emails servers are supported.

This resource also acts as an SMTP client, using `PLAIN` auth over TLS.  So you need an SMTP server that supports all that.

For development, we've been using [Amazon SES](https://aws.amazon.com/ses/) with its [SMTP support](http://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html)

## Source Configuration

### `source`:

#### Parameters

Within imap:

* `host`: *Required.* SMTP Host name
* `port`: *Required.* SMTP Port, must be entered as a string
* `username`: *Required.* Username to authenticate with
* `password`: *Required.* Password to authenticate with
* `inbox`: *Required.* Message inbox to watch for updates
* `skip_ssl_validation`: *Optional.* Whether or not to skip ssl validation.  true/false are valid options.  If omitted default is false.

Within smtp:

* `host`: *Required.* SMTP Host name
* `port`: *Required.* SMTP Port, must be entered as a string
* `anonymous`: *Optional.* Whether or not to require credential.  true/false are valid options.  If omitted default is false
* `username`: *Required, Conditionally.* Username to authenticate with.  Ignored if `anonymous: true`
* `password`: *Required, Conditionally.* Password to authenticate with.  Ignored if `anonymous: true`
* `skip_ssl_validation`: *Optional.* Whether or not to skip ssl validation.  true/false are valid options.  If omitted default is false
* `ca_cert`: *Optional.* Certificates content to verify servers with custom certificates. Only considered if `skip_ssl_validation` is `false`.

Within source:
* `from`: *Required.* Email Address to be sent from.
* `to`: *Required.Conditionally.* Array of email addresses to send email to.  Not required if job params contains a file reference that has to recipients.

An example source configuration is below.
```yaml
resources:
- name: send-an-email
  type: email
  source:
    imap:
      host: imap.example.com
      port: "587" # this must be a string
      username: a-user
      password: my-password
      inbox: "INBOX"
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
[automatically gets converted to a string](http://concourse.ci/fly-cli.html).
But for literals you need to surround it with quotes.

## Behavior

### `check`: Check for new emails

The provided `inbox` is scanned and the last 4 emails found will be returned.

### `in`: Pull in a specific email

Downloads the targetted email onto the file system. The email information is seperated into `subject`, `body`, `version`, `date` files and an `attachments` folder.

### `out`: Send an email

#### Parameters

* `headers`: *Optional.* Path to plain text file containing additional mail headers
* `subject`: *Optional.* Path to plain text file containing the subject. Either `subject` or `subject_text` required. `subject_text` takes precedence.
* `subject_text`: *Optional.* The subject as text. Either `subject` or `subject_text` required. `subject_text` takes precedence.
* `body`: *Optional.* Path to file containing the email body. Either `body` or `body_text` required. `body_text` takes precedence.
* `body_text`: *Optional.* The email body as text. Either `body` or `body_text` required. `body_text` takes precedence.
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

You can use the values below in any of the source files or text properties to access the corresponding metadata made available by concourse, as documented [here](http://concourse.ci/implementing-resources.html)

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

* Install `Go`.
* Install [Glide](https://github.com/Masterminds/glide), a dependency management tool for Go.
* Clone the repo:
  - `mkdir -p $(go env GOPATH)/src/github.com/pivotal-cf`
  - `cd $(go env GOPATH)/src/github.com/pivotal-cf`
  - `git clone git@github.com:pivotal-cf/email-resource.git`
* Install dependencies:
  - `cd email-resource`
  - `glide install`
  - `go build -o bin/check check/cmd/*.go`
  - `go build -o bin/in in/cmd/*.go`
  - `go build -o bin/out out/cmd/*.go`

To cross compile, set the `$GOOS` and `$GOARCH` environment variables.
For example: `GOOS=linux GOARCH=amd64 go build`.

## Testing

To run the unit tests, use `go test $(glide nv)`.
