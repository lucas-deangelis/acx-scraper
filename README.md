# ACX Scraper

Astral Codex Ten comment scraper.

## Usage

You will need Go installed on your machine.

```sh
$ go build
$ ./acx-scraper articles 
$ ./acx-scraper comments
```

`./acx-scraper articles` will fill the databaese with all the articles metadata. It does not download the articles themselves.

`./acx-scraper comments` will read the database, and download the comments of every article in the database.

There is a delay of 1 second between each request to Substack, which I consider scraping etiquette.

As the time of writing, there are more than 900 articles, so you can expect ~15 minutes of runtime, most of which is the 1 second delay. between requests. The final SQLite file is 1.5 Gb.

If you want to make it smaller you can remove the original JSON:

```
$ sqlite3 <database-file>
sqlite> ALTER TABLE comments DROP COLUMN OriginalJSON;
sqlite> ALTER TABLE articles DROP COLUMN OriginalJSON;
sqlite> VACUUM;
sqlite> .quit
```

The database without the original JSON is just 272 Mb.

## Dev notes

### Old version (writing the JSON, and then inserting the JSON in the database)

- Using transactions with `tx.Stmt` to transform a prepared statement to a transaction-specific prepared statement offered a humongus speedup. SQL went from being ~90% of the runtime (as per pprof) to ~25%
- now the runtime is dominated by `json.Unmarshal`, ~65%

### New version (writing directly in the database)

- Runtime seems to be mostly `time.Sleep(1 * time.Second)` (1 API call every second be respectful)
