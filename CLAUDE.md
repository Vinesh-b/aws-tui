## Testing
- Always run `make test` (`go test -race ./...`) and `make lint` before reporting done.
- UI code calls APIs from background goroutines, so fakes must be race-safe. Put a fake that tests change after the app starts in its own `<service>fake` package under the service's directory (for example `snsfake`), with private state behind one mutex. Configure it through constructor options (`New(WithX(...))`) and `Set*` methods, and read what was called through accessor methods. Never add exported or package-visible mutable fields to a fake.
- Anything a test shares with code that runs on another goroutine (the tview event loop, pollers, API calls) must be race-safe, including test helpers and closures. Never share a bare `strings.Builder`, `bytes.Buffer`, slice, map or counter. `internal/safetest` has `LogCapture` (give the app `logs.Logger()`), `Recorder[T]` (`Record` from a callback, `Calls`/`Count` in the test) and `Value[T]` (`Set`/`Get`).
- Fakes return copies from their accessors (`slices.Clone`, `maps.Clone`), never their internal slice or map or pointers to stored structs.
- After touching tests that run the app, repeat them with `go test -race -count=20 ./path/...`; a race that shows up once in CI usually reproduces that way.
- Use the internal/safetest helpers for shared state in fakes and test helpers. Never read counters outside the UI goroutine without synchronisation.
- Never use time.Sleep for synchronisation. Wait on specific, unique text or explicit signals, never generic strings like a single letter.
- Regression tests must fail without the fix. Confirm this before committing.
- For UI changes (themes, navigation, key bindings), state clearly what was NOT verified in a real terminal, and check every affected view type (FilterableList, TabView inner navigators, etc.).

## Following Existing Patterns
Before building a new view, tab, fake or editor, find the closest existing example (e.g. the logs view for tab views) and copy its structure. Do not invent new layouts. Editors and forms go inside the TabView as tabs, never as floating overlays. Fakes live in their own package and follow the Api/Calls() convention. Prefer fixed, explicit lists (e.g. standard Lambda metrics) over dynamic discovery unless asked otherwise.
Add under a ## Git & Commits section

## Scope Discipline
Only change what was requested. Do not add new lint checks, Makefile targets or CLAUDE.md rules unless asked. Never rewrite, amend or rebase commits that were already made or pushed. If splitting work into commits would require touching history, ask first. Keep the user's pre-existing uncommitted edits out of commits.
Add under a ## Testing section in CLAUDE.md, replacing or merging with any existing test guidance.
