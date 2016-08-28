package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"

	// Load the common drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	// Load sqlx over database/sql
	"github.com/jmoiron/sqlx"

	"github.com/StabbyCutyou/sqltocsv/converters"
)

type config struct {
	dbAdapter       string
	connString      string
	sqlQuery        string
	outputFile      string
	delimiter       string
	obfuscateFields string
	quoteFields     string
	quoteType       string
}

var delimiters = map[string]rune{
	"tab": rune('	'), //That's a tab in there, yo
	"comma": rune(','),
}

func main() {
	run(getConfig())
}

// run is broken out so that it's easier to test
func run(cfg *config) {
	// Get the connection to the DB
	db, err := sqlx.Open(cfg.dbAdapter, cfg.connString)
	if err != nil {
		log.Fatal(err)
	}

	// Run the initial query
	results, err := db.Queryx(cfg.sqlQuery)
	if err != nil {
		log.Fatal(err)
	}

	// Redirect the output to STDOUT
	// All log messages write to STDEER to make redirecting the output to a file
	// simpler
	csvWriter := csv.NewWriter(os.Stdout)
	if comma, ok := delimiters[cfg.delimiter]; ok {
		csvWriter.Comma = comma
	} else {
		log.Printf("Warning: No known delimiter for %s, defaulting to Comma", cfg.delimiter)
	}

	// Get our converter
	converter := converters.GetConverter(cfg.dbAdapter)

	count := 0
	// Stream the result set
	for results.Next() {
		row, err := results.SliceScan()
		if err != nil {
			log.Fatal(err)
		}
		// Only do this for the first line, aka the headers
		if count == 0 {
			cols, err := results.Columns()
			if err != nil {
				log.Fatal(err)
			}
			csvWriter.Write(cols)
		}

		rowStrings := make([]string, len(row))
		for i, col := range row {
			val, err := converter.ColumnToString(col)
			if err != nil {
				log.Fatal(err)
			}
			// TODO Inject quoting, obfuscating here
			rowStrings[i] = val
		}
		csvWriter.Write(rowStrings)
		count++
	}

	csvWriter.Flush()
	log.Printf("\nFinished processing %d lines\n", count)

}

func getConfig() *config {
	d := flag.String("d", "mysql", "The (d)atabase adapter to use")
	c := flag.String("c", "", "The (c)onnection string to use")
	q := flag.String("q", "", "The (q)uery to use")
	m := flag.String("m", "comma", "The deli(m)iter to use: 'comma' or 'tab'. Defaults to 'comma'")
	o := flag.String("o", "", "The fields to (o)bfuscate")
	w := flag.String("w", "", "The fields to (w)rap in quotes")
	t := flag.String("t", "double", "The (t)ype of quote to use with -w: 'single' or 'double'. Defaults to 'double'")

	flag.Parse()

	if *q == "" {
		log.Fatal("You must provide query via -q")
	}
	if *c == "" {
		log.Fatal("You must provide a connection string via -c")
	}

	return &config{
		dbAdapter:       *d,
		connString:      *c,
		sqlQuery:        *q,
		obfuscateFields: *o,
		delimiter:       *m,
		quoteFields:     *w,
		quoteType:       *t,
	}
}

//SELECT * FROM users WHERE created_at >= '2015-01-01 00:00:00' AND created_at < '2015-02-01 00:00:00'
