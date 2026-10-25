# aws-tui

A terminal UI for browsing AWS services, built with [tview](https://github.com/rivo/tview) and the AWS SDK for Go v2.

## Supported services

Lambda, CloudWatch (Logs, Logs Insights, Alarms, Metrics), DynamoDB, Step Functions, S3, Systems Manager (Parameter Store), EventBridge, CloudFormation, SQS, SNS and VPC.

## Requirements

- Go (see `go.mod` for the version)
- AWS credentials and region configured the usual way (environment variables, `~/.aws/config`, SSO, ...). The profile can be switched from inside the app.

## Build and run

```sh
make build        # outputs ./bin/aws-tui
./bin/aws-tui
./bin/aws-tui -version
```

Other targets: `make build-all` (linux, darwin, windows), `make test`, `make clean`.

## Settings

The theme chosen in the Settings page is saved and used the next time the app starts. It is kept in `settings.json` in the config directory of the user (`~/.config/aws-tui/` on Linux, `~/Library/Application Support/aws-tui/` on macOS, `%AppData%\aws-tui\` on Windows). If the file cannot be read, or names a theme that does not exist, the app starts with the default theme and logs why in the Debug Logs page.

To choose the directory proposed when saving a table to csv, set `downloadDir` in `settings.json`, for example `{"downloadDir": "~/Downloads"}`. Without it the working directory is used, and either way the directory follows the last file saved. Existing files are never overwritten.

## Project layout

```
cmd/aws-tui/        entry point (flags, AWS config loading)
internal/awsapi/    thin wrappers around the AWS SDK clients, one file per service
internal/errors/    shared error types
internal/settings/  the settings file that keeps the user's choices between runs
internal/utils/     small generic helpers
internal/ui/app/    wires the services together into the tview application
internal/ui/core/   reusable widgets, table infrastructure and app-wide state
internal/ui/services/  service picker, help and profile selection
internal/ui/<service>/ views and tables for one service
                       (cloudformation, cloudwatch, dynamodb, eventbridge,
                        lambda, s3, sfn, ssm, vpc)
```

## License

See [LICENSE](LICENSE).
