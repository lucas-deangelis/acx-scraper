package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/google/subcommands"
	_ "github.com/mattn/go-sqlite3"
)

type commentsJSON struct {
	Comments []comment
}

type comment struct {
	ID            int64
	PostID        int64 `json:"post_id"`
	UserID        int64 `json:"user_id"`
	Date          string
	Body          *string
	Name          string
	Deleted       bool
	AncestorPath  string `json:"ancestor_path"`
	ChildrenCount int64  `json:"children_count"`
	Children      []comment
}

type article struct {
	ID                      int64
	PublicationID           int64 `json:"publication_id"`
	Title                   string
	SocialTitle             string `json:"social_title"`
	Slug                    string
	PostDate                string `json:"post_date"`
	Audience                string
	WriteCommentPermissions string `json:"write_comment_permissions"`
	CanonicalURL            string `json:"canonical_url"`
	CoverImage              string `json:"cover_image"`
	Description             string
	WordCount               int64
	CommentCount            int64 `json:"comment_count"`
	ChildCommentCount       int64 `json:"child_comment_count"`
}

type articlesCmd struct {
	database   string
	cpuProfile bool
}

func (*articlesCmd) Name() string {
	return "articles"
}

func (*articlesCmd) Synopsis() string {
	return "Get all the articles from ACX."
}

func (*articlesCmd) Usage() string {
	return `articles [-d/-database <database_name>] -c/-cpuProfile
	Get all the articles from ACX, write it in the database.
`
}

func (a *articlesCmd) SetFlags(f *flag.FlagSet) {
	date := time.Now().Format("2006-01-02")
	dbName := "acx-comments_" + date + ".db"
	usage := "sqlite database name. The default name is acx-comments_YYYY-MM-DD.db"
	f.StringVar(&a.database, "database", dbName, usage)
	f.StringVar(&a.database, "d", dbName, usage)
	f.BoolVar(&a.cpuProfile, "c", false, "write cpu profile")
	f.BoolVar(&a.cpuProfile, "cpuProfile", false, "write cpu profile")
}

func (a *articlesCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if a.cpuProfile {
		a, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(a); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	getArticles(a.database)
	return subcommands.ExitSuccess
}

type commentsCmd struct {
	database   string
	cpuProfile bool
}

func (*commentsCmd) Name() string {
	return "comments"
}

func (*commentsCmd) Synopsis() string {
	return "Get all the comments of the ACX articles in the database."
}

func (*commentsCmd) Usage() string {
	return `comments [-d/-database <database_name>] -c/-cpuProfile
	Read the database to get all the articles, get the comments for each articles,
	insert them in the database.
`
}

func (c *commentsCmd) SetFlags(f *flag.FlagSet) {
	date := time.Now().Format("2006-01-02")
	dbName := "acx-comments_" + date + ".db"
	usage := "sqlite database name. The default name is acx-comments_YYYY-MM-DD.db"
	f.StringVar(&c.database, "database", dbName, usage)
	f.StringVar(&c.database, "d", dbName, usage)
	f.BoolVar(&c.cpuProfile, "c", false, "write cpu profile")
	f.BoolVar(&c.cpuProfile, "cpuProfile", false, "write cpu profile")
}

func (c *commentsCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.cpuProfile {
		a, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(a); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	getComments(c.database)
	return subcommands.ExitSuccess
}

type bodiesCmd struct {
	database   string
	cpuProfile bool
}

func (*bodiesCmd) Name() string {
	return "bodies"
}

func (*bodiesCmd) Synopsis() string {
	return "Get all the articles bodies from ACX."
}

func (*bodiesCmd) Usage() string {
	return `bodies [-d/-database <database_name>] -c/-cpuProfile
	Read the database to get all the articles, get article bodies for each articles,
	insert them in the database.
`
}

func (c *bodiesCmd) SetFlags(f *flag.FlagSet) {
	date := time.Now().Format("2006-01-02")
	dbName := "acx-comments_" + date + ".db"
	usage := "sqlite database name. The default name is acx-comments_YYYY-MM-DD.db"
	f.StringVar(&c.database, "database", dbName, usage)
	f.StringVar(&c.database, "d", dbName, usage)
	f.BoolVar(&c.cpuProfile, "c", false, "write cpu profile")
	f.BoolVar(&c.cpuProfile, "cpuProfile", false, "write cpu profile")
}

func (c *bodiesCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.cpuProfile {
		a, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(a); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	getBodies(c.database)
	return subcommands.ExitSuccess
}

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&articlesCmd{}, "")
	subcommands.Register(&commentsCmd{}, "")
	subcommands.Register(&bodiesCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

func flattenComments(comments []comment) []comment {
	if len(comments) == 0 {
		return []comment{}
	}

	var output []comment

	for _, c := range comments {
		output = append(output, c)
		output = append(output, flattenComments(c.Children)...)
	}

	return output
}

func insertComments(db *sql.DB, comments []comment) error {
	stmt, err := db.Prepare("INSERT INTO comments (ID, PostID, UserID, Date, Body, Name, AncestorPath, ChildrenCount, OriginalJSON) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txStmt := tx.Stmt(stmt)

	for _, c := range comments {
		originalJSON, err := json.Marshal(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		_, err = txStmt.Exec(c.ID, c.PostID, c.UserID, c.Date, c.Body, c.Name, c.AncestorPath, c.ChildrenCount, originalJSON)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func getBodies(databaseName string) {
	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		log.Fatalf("Failed to open database %s: %v", databaseName, err)
	}
	defer db.Close()

	// commentsSchema := `ALTER TABLE articles ADD COLUMN BodyHTML TEXT;`

	// _, err = db.Exec(commentsSchema)
	// if err != nil {
	// 	log.Fatalf("Failed to alter 'comments' table: %v", err)
	// }

	rows, err := db.Query(`SELECT Slug FROM articles;`)
	if err != nil {
		log.Fatalf("Failed to get the articles IDs: %v", err)
	}
	defer rows.Close()
	var articlesSlugs []string
	for rows.Next() {
		var articleSlug string
		if err := rows.Scan(&articleSlug); err != nil {
			log.Fatal(err)
		}
		articlesSlugs = append(articlesSlugs, articleSlug)
	}

	for _, articleSlug := range articlesSlugs {
		url := fmt.Sprintf("https://www.astralcodexten.com/api/v1/posts/%s", articleSlug)
		fmt.Println(url)

		// Create a new request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		// Add the "Accept: application/json" header
		req.Header.Add("Accept", "application/json")

		// Send the request using http.Client
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		var articleResponse struct {
			BodyHTML string `json:"body_html"`
		}
		err = json.Unmarshal(body, &articleResponse)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		_, err = db.Exec(`UPDATE articles SET BodyHTML = ? WHERE Slug = ?;`, articleResponse.BodyHTML, articleSlug)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		time.Sleep(1 * time.Second)
	}
}

func getComments(databaseName string) {
	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		log.Fatalf("Failed to open database %s: %v", databaseName, err)
	}
	defer db.Close()

	commentsSchema := `
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
	);`

	_, err = db.Exec(commentsSchema)
	if err != nil {
		log.Fatalf("Failed to create 'comments' table: %v", err)
	}

	rows, err := db.Query(`SELECT ID FROM articles;`)
	if err != nil {
		log.Fatalf("Failed to get the articles IDs: %v", err)
	}
	defer rows.Close()
	var articlesIDs []int64
	for rows.Next() {
		var articleID int64
		if err := rows.Scan(&articleID); err != nil {
			log.Fatal(err)
		}
		articlesIDs = append(articlesIDs, articleID)
	}

	for _, articleID := range articlesIDs {

		url := fmt.Sprintf("https://www.astralcodexten.com/api/v1/post/%d/comments?token=&all_comments=true&sort=oldest_first", articleID)
		fmt.Println(url)
		res, err := http.Get(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		var commentFile commentsJSON
		err = json.Unmarshal(body, &commentFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		flatComments := flattenComments(commentFile.Comments)

		err = insertComments(db, flatComments)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		time.Sleep(1 * time.Second)
	}
}

// noArticles is the JSON returned by the Substack API when there are no articles at that offset.
const noArticles = "[]"

// getArticles downloads all the article metadata from ACX, 12 by 12, and store it in JSON files in an "articles" folder.
// The JSON files are named "article_offset_N", with N being the offset used in the query.
func getArticles(databaseName string) {
	// First open/create the database and the 'articles' table.
	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		log.Fatalf("Failed to open the database %s: %v", databaseName, err)
	}
	defer db.Close()

	articlesSchema := `
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
	);`

	_, err = db.Exec(articlesSchema)
	if err != nil {
		log.Fatalf("Failed to create 'articles' table: %v", err)
	}

	// Loop on the articles until there are no left, and store them in the database.
	offset := 0
	baseURL := `https://www.astralcodexten.com/api/v1/archive?sort=new&search=&offset=%d&limit=12`

	for {
		// Query the API, get the articles, read the body.
		url := fmt.Sprintf(baseURL, offset)
		fmt.Println(url)
		res, err := http.Get(url)
		if err != nil {
			log.Fatalf("Failed to get '%s': %v", baseURL, err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Failed to read the body of '%s': %v", baseURL, err)
		}

		if string(body) == noArticles {
			break
		}

		// Read the body as JSON, store it in the database.
		var articles []article
		var articlesJSON []interface{}

		err = json.Unmarshal(body, &articles)
		if err != nil {
			log.Fatalf("Failed to unmarshal body into struct of '%s': %v", baseURL, err)
		}

		err = json.Unmarshal(body, &articlesJSON)
		if err != nil {
			log.Fatalf("Failed to unmarshal body of '%s': %v", baseURL, err)
		}

		stmt, err := db.Prepare("INSERT INTO articles (ID, PublicationID, Title, SocialTitle, Slug, PostDate, Audience, WriteCommentPermissions, CanonicalURL, CoverImage, Description, WordCount, CommentCount, ChildCommentCount, OriginalJSON) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			log.Fatalf("Failed to prepare statement: %v", err)
		}

		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("Failed to begin transaction: %v", err)
		}

		txStmt := tx.Stmt(stmt)

		for i, article := range articles {
			firstArticleAsByte, err := json.Marshal(articlesJSON[i])
			if err != nil {
				log.Fatalf("Failed to marshal article %d: %v", i, err)
			}
			firstArticleAsString := string(firstArticleAsByte)

			txStmt.Exec(article.ID, article.PublicationID, article.Title, article.SocialTitle, article.Slug, article.PostDate, article.Audience, article.WriteCommentPermissions, article.CanonicalURL, article.CoverImage, article.Description, article.WordCount, article.CommentCount, article.ChildCommentCount, firstArticleAsString)
		}

		err = tx.Commit()
		if err != nil {
			log.Fatalf("Failed to commit statement: %v", err)
		}

		offset += 12
		time.Sleep(1 * time.Second)
	}

	res, err := db.Query("SELECT COUNT(ID) FROM articles;")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer res.Close()

	res.Next()
	var articlesNumber int
	err = res.Scan(&articlesNumber)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Printf("%d articles found\n", articlesNumber)
	}
}
