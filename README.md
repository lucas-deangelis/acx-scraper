# ACX Scraper

Download all the comments from the [Astral Codex Ten substack](https://www.astralcodexten.com/) and put them in a SQLite database.

If you want to download SQLite files containing already the comments, see the Github releases. I try to update it every 6 month on average.

## Prerequisites

You will need Go installed on your machine. [Here](https://go.dev/dl/) is the official installation page. To build the program:

```sh
$ go build
```

## Usage

To download all the comments:

```sh
$ ./acx-scraper articles 
$ ./acx-scraper comments
```

`./acx-scraper articles` will fill the database with all the articles metadata.

`./acx-scraper comments` will read the database, and download the comments of every article in the database.

There is a delay of 1 second between each request to Substack, which I consider scraping etiquette. As the time of writing, there are more than 1050 articles. At 1 second for each article, it'll take 15-20 minutes. The final SQLite file is 1.5 Gb.

If you want to make it smaller you can remove the original JSON:

```
$ sqlite3 <database-file>
sqlite> ALTER TABLE comments DROP COLUMN OriginalJSON;
sqlite> ALTER TABLE articles DROP COLUMN OriginalJSON;
sqlite> VACUUM;
sqlite> .quit
```

The database without the original JSON is just 272 Mb.

## SQLite database

Articles:

```sql
CREATE TABLE IF NOT EXISTS articles (
    ID                      INTEGER PRIMARY KEY,
    PublicationID           INTEGER NOT NULL,
    Title                   TEXT NOT NULL,
    SocialTitle             TEXT NOT NULL,
    Slug                    TEXT UNIQUE NOT NULL,
    PostDate                TEXT NOT NULL,
    Audience                TEXT NOT NULL,
    WriteCommentPermissions TEXT NOT NULL,
    CanonicalURL            TEXT NOT NULL,
    CoverImage              TEXT NOT NULL,
    Description             TEXT NOT NULL,
    WordCount               INTEGER NOT NULL,
    CommentCount            INTEGER NOT NULL,
    ChildCommentCount       INTEGER NOT NULL,
    OriginalJSON			TEXT NOT NULL
);
```

Comments:

```sql
CREATE TABLE IF NOT EXISTS comments (
    ID INTEGER PRIMARY KEY,
    PostID INTEGER,
    UserID INTEGER,
    Date TEXT,
    Body TEXT,
    Name TEXT,
    AncestorPath TEXT,
    ChildrenCount INTEGER,
    OriginalJSON TEXT NOT NULL
);
```

## Dev notes

### Old version (writing the JSON, and then inserting the JSON in the database)

- Using transactions with `tx.Stmt` to transform a prepared statement to a transaction-specific prepared statement offered a humongus speedup. SQL went from being ~90% of the runtime (as per pprof) to ~25%.
- Now ~65% of the runtime is `json.Unmarshal`

### New version (writing directly in the database)

- Runtime seems to be mostly `time.Sleep(1 * time.Second)` (1 API call every second to be respectful)
