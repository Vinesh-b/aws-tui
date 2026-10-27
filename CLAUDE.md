# Testing

- Run tests with `make test` (`go test -race ./...`). CI uses the same command; a bare `go test` misses data races.
- UI code calls APIs from background goroutines, so fakes must be race-safe. Put a fake that tests change after the app starts in its own `<service>fake` package under the service's directory (for example `snsfake`), with private state behind one mutex. Configure it through constructor options (`New(WithX(...))`) and `Set*` methods, and read what was called through accessor methods. Never add exported or package-visible mutable fields to a fake.
- Never sleep in tests. Use `uitest.Eventually` / `uitest.Await*` and touch widgets only inside `uitest.OnUI`.
- Run `make lint` before committing.
