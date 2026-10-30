# Testing

- Run tests with `make test` (`go test -race ./...`). CI uses the same command; a bare `go test` misses data races.
- UI code calls APIs from background goroutines, so fakes must be race-safe. Put a fake that tests change after the app starts in its own `<service>fake` package under the service's directory (for example `snsfake`), with private state behind one mutex. Configure it through constructor options (`New(WithX(...))`) and `Set*` methods, and read what was called through accessor methods. Never add exported or package-visible mutable fields to a fake.
- Never sleep in tests (`make lint` rejects an unmarked `time.Sleep`). Wait for something to happen with `uitest.Eventually` / `uitest.Await*`, and check that something does *not* happen or change with `uitest.Consistently`. Pass a short description to both so a timeout says what was being waited for. Touch widgets only inside `uitest.OnUI`.
- Anything a test shares with code that runs on another goroutine (the tview event loop, pollers, API calls) must be race-safe, including test helpers and closures. Never share a bare `strings.Builder`, `bytes.Buffer`, slice, map or counter. `internal/safetest` has `LogCapture` (give the app `logs.Logger()`), `Recorder[T]` (`Record` from a callback, `Calls`/`Count` in the test) and `Value[T]` (`Set`/`Get`).
- Fakes return copies from their accessors (`slices.Clone`, `maps.Clone`), never their internal slice or map or pointers to stored structs.
- After touching tests that run the app, repeat them with `go test -race -count=20 ./path/...`; a race that shows up once in CI usually reproduces that way.
- Run `make lint` before committing.
