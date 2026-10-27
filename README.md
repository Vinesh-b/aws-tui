# aws-tui

A terminal UI for browsing AWS services, built with [tview](https://github.com/rivo/tview) and the AWS SDK for Go v2.

## Supported services

Lambda, CloudWatch (Logs, Logs Insights, Alarms, Metrics), DynamoDB, IAM (Roles, Users, Policies), Step Functions, S3, Systems Manager (Parameter Store), EventBridge, CloudFormation, SQS, SNS and VPC.

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

## Navigation

Alt+Left and Alt+Right go back and forward through the services and pages that were shown, like the history of a browser. Opening a link to another service, such as the log group of a lambda, can be gone back from. Switching profile or region starts the history again, as the pages are loaded again.

## Settings

The theme chosen in the Settings page is saved and used the next time the app starts. It is kept in `settings.json` in the config directory of the user (`~/.config/aws-tui/` on Linux, `~/Library/Application Support/aws-tui/` on macOS, `%AppData%\aws-tui\` on Windows). If the file cannot be read, or names a theme that does not exist, the app starts with the default theme and logs why in the Debug Logs page.

To choose the directory proposed when saving a table to csv, set `downloadDir` in `settings.json`, for example `{"downloadDir": "~/Downloads"}`. Without it the working directory is used, and either way the directory follows the last file saved. Existing files are never overwritten.

To change a key, set `keyBindings` in `settings.json` to the name of the binding and the key, for example `{"keyBindings": {"reset": "R", "clearTable": "Ctrl+D", "pageChangeModKey": "Ctrl"}}`. A key is one character (or `space`) for most bindings, a name such as `Esc`, `Enter`, `PgUp`, `Ctrl+X` or `Ctrl+Space` for the others, and `Alt`, `Ctrl` or `Shift` for the modifiers. The names are `help`, `quit`, `escape`, `toggleServicesMenu`, `toggleServicePages`, `reset`, `loadMoreData`, `clearTable`, `saveTable`, `done`, `find`, `pageChangeModKey`, `pageForward`, `pageBack`, `historyModKey`, `historyBack`, `historyForward`, `viewFocusUp`, `viewFocusDown`, `viewFocusLeft`, `viewFocusRight`, `viewResizeModKey`, `viewResizeReset`, `moveUp`, `moveDown`, `moveLeft`, `moveRight`, `moveLineStart`, `moveLineEnd`, `moveTop`, `moveBottom`, `nextSearch`, `prevSearch`, `formFocusNext`, `formFocusPrev`, `tableScan`, `tableQuery`, `mark`, `removeItem`, `textCopy`, `toggleRawView`, `openLink`, `textViewPageUp`, `textViewPageDown`, `textViewWordRight`, `textViewWordLeft`, `textViewUndo`, `textViewRedo`, `textViewVisualMode` and `textViewInsertMode`. The help views show the keys in use. A key that is unknown, cannot be read, or is already used by another binding in the same view is ignored and logged in the Debug Logs page. The number keys that switch tabs, and the keys of text inputs such as Shift+Arrows to select, cannot be changed. Terminals without the newer keyboard protocols, such as tmux without `extended-keys`, send Ctrl+H as Backspace, so Backspace also moves to the pane on the left outside text inputs.

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
                        iam, lambda, s3, sfn, sns, sqs, ssm, vpc)
```

## License

See [LICENSE](LICENSE).
