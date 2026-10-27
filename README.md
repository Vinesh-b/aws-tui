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

## Keys

The keys follow Vim where they can. `?` lists the keys of the selected pane, and the Help page (`:help`) lists them all as they are bound. Every key in the tables below can be changed on the Keys page of the settings, or in the settings file under the name in the last column (see [Key settings](#key-settings)).

### Moving around

| Key | Does | Setting |
|---|---|---|
| `Ctrl+Space` | Show or hide the services menu | `toggleServicesMenu` |
| `Ctrl+P` | Show or hide the pages of the service | `toggleServicePages` |
| `gT` / `gt` | Go to the page on the left / right | `pageBackSequence` / `pageForwardSequence` |
| `Alt+Left` / `Alt+Right` | Go back / forward through the pages shown | `historyModKey` + `historyBack` / `historyForward` |
| `Ctrl+K` / `Ctrl+J` | Move to the pane above / below | `viewFocusUp` / `viewFocusDown` |
| `Ctrl+H` / `Ctrl+L` | Move to the pane on the left / right | `viewFocusLeft` / `viewFocusRight` |
| `Alt+h` `Alt+j` `Alt+k` `Alt+l` | Move the split between two panes | `viewResizeModKey` + the move keys |
| `Alt+0` | Put the splits back | `viewResizeReset` |
| `Esc` | Hide a floating view, such as a menu or a search | `escape` |
| `?` | Show the keys of the selected pane | `help` |
| `1`–`9` | Go to a tab, in views with tabs | — |

Going back and forward works like the history of a browser, and also across services: after following a link, such as the log group of a lambda, going back returns to where the link was. Switching profile or region starts the history again, as the pages are loaded again.

### Tables

| Key | Does | Setting |
|---|---|---|
| `j` / `k` | Move down / up a row | `moveDown` / `moveUp` |
| `gg` / `G` | Go to the first / last row | `moveTopSequence` / `moveBottom` |
| `PgDn` / `PgUp`, `Ctrl+F` / `Ctrl+B` | Move a page down / up | — |
| `Enter` | Open the selected item | `done` |
| `r` | Load the table again | `reset` |
| `Ctrl+X` | Clear the table | `clearTable` |
| `/` | Search the table | `find` |
| `n` / `N` | Go to the next / previous match | `nextSearch` / `prevSearch` |
| `y` | Copy the text of the cell | `textCopy` |
| `q` | Query the table, such as a DynamoDB table, or the marked log groups with Insights | `tableQuery` |
| `s` | Scan the DynamoDB table | `tableScan` |
| `m` | Mark a log group to query with Insights | `mark` |
| `x`, `dd` | Unmark a log group, delete a saved query, or delete a DynamoDB item once it is confirmed | `removeItem`, `removeItemSequence` |

Tables that load their rows a page at a time, such as log events or S3 objects, load the next page when the cursor moves down from the last row (with `j`, Down, PgDn or `G`). The cursor stays where it was, so moving down again goes on into the new rows.

### Details

| Key | Does | Setting |
|---|---|---|
| `y` | Copy the value of the selected row | `textCopy` |
| `J` | Switch between the details and the API response as JSON | `toggleRawView` |
| `gx` | Open the resource an underlined value links to | `openLinkSequence` |

### Text views

| Key | Does | Setting |
|---|---|---|
| `h` `j` `k` `l` | Move left, down, up, right | `moveLeft`, `moveDown`, `moveUp`, `moveRight` |
| `w` / `b` | Move a word forward / back | `textViewWordRight` / `textViewWordLeft` |
| `^` / `$` | Move to the start / end of the line | `moveLineStart` / `moveLineEnd` |
| `gg` / `G` | Move to the top / bottom | `moveTopSequence` / `moveBottom` |
| `Ctrl+U` / `Ctrl+D` | Move a page up / down | `textViewPageUp` / `textViewPageDown` |
| `/`, `n` / `N` | Search, and go to the next / previous match | `find`, `nextSearch` / `prevSearch` |
| `v` | Select text in visual mode | `textViewVisualMode` |
| `y` | Copy the selected text, in visual mode | `textCopy` |
| `Esc` | Go back to normal mode | `escape` |
| `i`, `u` / `Ctrl+R` | Insert mode, undo / redo; text views are read-only, so these change nothing yet | `textViewInsertMode`, `textViewUndo` / `textViewRedo` |

In forms, `Down` and `Up` move to the next and previous field (`formFocusNext`, `formFocusPrev`). Text inputs keep their own keys: the arrows move the cursor, `Shift` + arrows select text, and `Ctrl+Q` copies it. These can't be changed.

### Sequences

`gg`, `gt`, `gT`, `gx` and `dd` are sequences: keys typed one after the other, as in Vim.

- A sequence waits a second for its next key.
- A key that doesn't continue a sequence drops the keys typed before it, and then does what it does on its own.
- While a sequence starts with a key, that key does nothing on its own: `g` alone waits for the rest of `gg`, `gt`, `gT` or `gx`.
- Sequences never take keys typed into a text input.
- `gg` does what the key `moveTop` (`g`) does in tables and text views. That key is only reached on its own when `moveTopSequence` is turned off or moved to other keys.

### Commands

`:` opens a command line at the bottom of the screen, outside text inputs (setting: `commandLine`).

| Command | Does |
|---|---|
| `:q`, `:quit` | Quit |
| `:w [file]`, `:write [file]` | Save the selected table to a csv file, at the path given or else in the download directory, named after the table |
| `:<service>` | Go to the service |

A service is named by its name in lower case without spaces, such as `:lambda` or `:cloudwatchlogs`, or by a short name: `:logs`, `:alarms`, `:metrics`, `:ddb`, `:sfn`, `:s3`, `:ssm`, `:cfn` or `:profile`. The start of a name is enough when only one service has it, and `:help` and `:settings` open those pages.

Commands are completed as they are typed: Tab puts the selected completion in, and Enter runs it. What a command did, or why it failed, stays in the line until the next key.

## Settings

The Settings page (`:settings`) has a page for each section of the settings file. Changes apply straight away and are saved.

| Page | Section of `settings.json` | Holds |
|---|---|---|
| Theme | `theme` | The colours of the app |
| Files | `files` | `downloadDir`, the directory tables are saved to with `:w` |
| Keys | `keys` | The keys that differ from their default |

On the Files and Keys pages, Enter changes the selected setting, and `x` or `dd` puts back its default. Changed settings are shown in another colour, and `/` searches them.

The settings file is `settings.json` in the config directory of the user:

- Linux: `~/.config/aws-tui/`
- macOS: `~/Library/Application Support/aws-tui/`
- Windows: `%AppData%\aws-tui\`

It looks like this, with only the settings that were changed:

```json
{
  "theme": "Nord",
  "files": {
    "downloadDir": "~/Downloads"
  },
  "keys": {
    "reset": "R",
    "clearTable": "Ctrl+D",
    "historyModKey": "Ctrl",
    "pageForwardSequence": "]p",
    "openLinkSequence": ""
  }
}
```

If the file cannot be read, the app starts with the defaults and logs why in the Debug Logs page.

### Theme

The theme is used the next time the app starts too. A theme that does not exist is replaced by the default one.

### Files

`downloadDir` is the directory tables are saved to as csv files. Without it the working directory is used, and either way the directory follows the last file saved while the app runs. Existing files are never overwritten.

### Key settings

`keys` maps the setting name of a key, from the tables in [Keys](#keys), to the key to use. What a setting takes depends on the kind of key:

- **A character**, for most keys: one character, or `space`.
- **A special key**: a name such as `Esc`, `Enter`, `Tab`, `Backspace`, `PgUp`, `PgDn`, `Up`, `Down`, `Left`, `Right`, `Home`, `End`, `Ctrl+X` or `Ctrl+Space`.
- **A modifier** (the settings ending in `ModKey`): `Alt`, `Ctrl` or `Shift`.
- **A sequence** (the settings ending in `Sequence`): two or more characters, or `""` to turn the sequence off.

A key can't be used when it can't be read, or is already used by another key of the same view. A key can't be the first key of a sequence either, as it would always wait for the rest of the sequence. The Keys page says why and keeps the old key; in the file, the setting is ignored and logged in the Debug Logs page, as are names that are not keys.

### Terminals

Copied text, with `y` or `Ctrl+Q`, goes to the clipboard of the terminal with the OSC 52 escape sequence, which works over SSH and in tmux, and to the clipboard of the system with xclip, xsel, wl-copy or pbcopy when one of them is installed. Most terminals take OSC 52, though some need it turned on in their settings; tmux passes it on with `set -g set-clipboard on`.

Terminals without the newer keyboard protocols, such as tmux without `extended-keys`, send `Ctrl+H` as Backspace and `Ctrl+M` as Enter. The services menu is on `Ctrl+Space` for that reason, and Backspace moves to the pane on the left like `Ctrl+H`, except in text inputs.

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
