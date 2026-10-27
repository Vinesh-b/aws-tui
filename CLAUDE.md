# Testing

- Run tests with `make test` (`go test -race ./...`). CI uses the same command; a bare `go test` misses data races.
- UI code calls APIs from background goroutines, so fakes must be race-safe. Use `internal/ui/sns/snsfake`: configure it with options (`snsfake.New(snsfake.WithTopics(...))`) or the `Set*` methods, and read calls with `Calls()` / `CallsWithPrefix()`. Never add exported or package-visible mutable fields to a fake.
- Never sleep in tests. Use `uitest.Eventually` / `uitest.Await*` and touch widgets only inside `uitest.OnUI`.
- Run `make lint` before committing.
