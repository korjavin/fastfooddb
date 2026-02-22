package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/korjavin/fastfooddb/internal/importer"
)

func main() {
	dump := flag.String("dump", "", "path to gzip-compressed JSONL dump (required)")
	out := flag.String("out", "", "output data directory (required)")
	verbose := flag.Bool("v", false, "print progress every 100k products")
	flag.Parse()

	if *dump == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: fastfooddb-importer -dump <path> -out <dir> [-v]")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting import", "dump", *dump, "out", *out)

	m, err := importer.Import(*dump, *out, *verbose)
	if err != nil {
		slog.Error("import failed", "error", err)
		os.Exit(1)
	}

	slog.Info("import complete",
		"products", m.ProductCount,
		"indexed", m.IndexedCount,
		"skipped", m.SkippedCount,
		"build_time", m.BuildTime,
	)
	fmt.Printf("Output: %s\n  Products stored : %d\n  Names indexed   : %d\n  Skipped         : %d\n",
		*out, m.ProductCount, m.IndexedCount, m.SkippedCount)

	if len(m.SkipReasons) > 0 {
		fmt.Println("  Skip reasons:")
		keys := make([]string, 0, len(m.SkipReasons))
		for k := range m.SkipReasons {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("    %-20s: %d\n", k, m.SkipReasons[k])
		}
	}
}
