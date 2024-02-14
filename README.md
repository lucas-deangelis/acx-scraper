# Uriage

Astral Codex Ten comment scrapper.

## Notes

### Old version (writing the JSON, and then inserting the JSON in the database)

- Using transactions with `tx.Stmt` to transform a prepared statement to a transaction-specific prepared statement offered a humongus speedup. SQL went from being ~90% of the runtime (as per pprof) to ~25%
- now the runtime is dominated by `json.Unmarshal`, ~65%

### New version (writing directly in the database)

- Runtime seems to be mostly `time.Sleep(1 * time.Second)` (1 API call every second be respectful)
