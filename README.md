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

Tables that load their rows a page at a time, such as log events or S3 objects, load the next page when the cursor moves down from the last row, with `j`, Down, PgDn or `G`. The cursor stays where it was, so moving down again goes on into the new rows.

## Commands

As in Vim, `:` opens a command line at the bottom, outside text inputs. `:q` quits, `:w` saves the selected table to a csv file, at the path given or else in the download directory, named after the table, and the name of a service goes to it, such as `:lambda`, `:cloudwatchlogs` or a short name: `:logs`, `:alarms`, `:metrics`, `:ddb`, `:sfn`, `:s3`, `:ssm`, `:cfn` and `:profile`. The start of a name is enough when only one service has it. Commands are completed as they are typed; Tab puts a completion in, and Enter runs it. What a command did, or why it failed, stays in the line until the next key.

## Settings

The theme chosen in the Settings page is saved and used the next time the app starts. It is kept in `settings.json` in the config directory of the user (`~/.config/aws-tui/` on Linux, `~/Library/Application Support/aws-tui/` on macOS, `%AppData%\aws-tui\` on Windows). If the file cannot be read, or names a theme that does not exist, the app starts with the default theme and logs why in the Debug Logs page.

To choose the directory proposed when saving a table to csv, set `downloadDir` in `settings.json`, for example `{"downloadDir": "~/Downloads"}`. Without it the working directory is used, and either way the directory follows the last file saved. Existing files are never overwritten.

To change a key, set `keyBindings` in `settings.json` to the name of the binding and the key, for example `{"keyBindings": {"reset": "R", "clearTable": "Ctrl+D", "historyModKey": "Ctrl"}}`. A key is one character (or `space`) for most bindings, a name such as `Esc`, `Enter`, `PgUp`, `Ctrl+X` or `Ctrl+Space` for the others, and `Alt`, `Ctrl` or `Shift` for the modifiers. The names are `help`, `commandLine`, `escape`, `toggleServicesMenu`, `toggleServicePages`, `reset`, `clearTable`, `done`, `find`, `historyModKey`, `historyBack`, `historyForward`, `viewFocusUp`, `viewFocusDown`, `viewFocusLeft`, `viewFocusRight`, `viewResizeModKey`, `viewResizeReset`, `moveUp`, `moveDown`, `moveLeft`, `moveRight`, `moveLineStart`, `moveLineEnd`, `moveTop`, `moveBottom`, `nextSearch`, `prevSearch`, `formFocusNext`, `formFocusPrev`, `tableScan`, `tableQuery`, `mark`, `removeItem`, `textCopy`, `toggleRawView`, `textViewPageUp`, `textViewPageDown`, `textViewWordRight`, `textViewWordLeft`, `textViewUndo`, `textViewRedo`, `textViewVisualMode`, `textViewInsertMode`, `moveTopSequence`, `pageForwardSequence`, `pageBackSequence`, `openLinkSequence` and `removeItemSequence`. The help views show the keys in use. A key that is unknown, cannot be read, or is already used by another binding in the same view is ignored and logged in the Debug Logs page. The sequences are keys typed one after the other, like the commands of Vim: `gg` goes to the top, `gt` and `gT` to the page on the right and left, and `gx` opens a link and `dd`, like `x`, unmarks a log group or deletes a saved query or, after it is confirmed, a DynamoDB item. Each is set as its keys, such as `{"keyBindings": {"pageForwardSequence": "]p"}}`, or as `""` to turn it off. A sequence waits a second for its next key, and a key that ends no sequence drops the keys typed before it. While a sequence starts with a key, that key does nothing on its own. The number keys that switch tabs, and the keys of text inputs such as Shift+Arrows to select, cannot be changed. Terminals without the newer keyboard protocols, such as tmux without `extended-keys`, send Ctrl+H as Backspace, so Backspace also moves to the pane on the left outside text inputs.

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
