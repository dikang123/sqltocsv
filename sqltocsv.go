package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	// Load the common drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	// Load sqlx over database/sql
	"github.com/jmoiron/sqlx"

	"github.com/StabbyCutyou/sqltocsv/converters"
)

// Config is
type Config struct {
	dbAdapter       string
	connString      string
	sqlQuery        string
	outputFile      string
	delimeter       string
	obfuscateFields string
	quoteFields     string
	quoteType       string
}

func main() {
	cfg := getConfig()
	db, err := sqlx.Open(cfg.dbAdapter, cfg.connString)
	if err != nil {
		log.Fatal(err)
	}

	results, err := db.Queryx(cfg.sqlQuery)
	if err != nil {
		log.Fatal(err)
	}

	output := os.Stdout

	csvWriter := csv.NewWriter(output)
	// If I ever need to support more than tabs/commas, this needs improving
	if cfg.delimeter == "tab" {
		csvWriter.Comma = 0x0009
	}

	converter := converters.GetConverter(cfg.dbAdapter)

	count := 0
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
		// It seems for mysql, the case is always []byte of a string?
		for i, col := range row {
			val, err := converter.ColumnToString(col)
			if err != nil {
				log.Fatal(err)
			}
			// Inject quoting, obfuscating here
			rowStrings[i] = val
		}
		csvWriter.Write(rowStrings)
		count++
	}

	csvWriter.Flush()
	fmt.Printf("\nFinished processing %d lines\n", count)
}

func getConfig() *Config {
	d := flag.String("d", "mysql", "The (d)atabase adapter to sue")
	c := flag.String("c", "", "The (c)onnection string to use")
	q := flag.String("q", "", "The (q)uery to use")
	m := flag.String("m", "comma", "The deli(m)eter to use: 'comma' or 'tab'. Defaults to 'comma'")
	o := flag.String("o", "", "The fields to (o)bfuscate")
	w := flag.String("w", "", "The fields to (w)rap in quotes")
	t := flag.String("t", "double", "The (t)ype of quote to use with -w: 'single' or 'double'. Defaults to 'double'")

	flag.Parse()

	if q == nil {
		log.Fatal("You must provide query via -q")
	}
	if c == nil {
		log.Fatal("You must provide a connection string via -c")
	}

	return &Config{
		dbAdapter:       *d,
		connString:      *c,
		sqlQuery:        *q,
		obfuscateFields: *o,
		delimeter:       *m,
		quoteFields:     *w,
		quoteType:       *t,
	}
}

//SELECT * FROM users WHERE created_at >= '2015-01-01 00:00:00' AND created_at < '2015-02-01 00:00:00'
