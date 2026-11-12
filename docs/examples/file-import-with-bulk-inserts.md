# File Import with Bulk Inserts

**Problem:** You import a large CSV or line-based file. Some rows are not valid. You want to insert the valid rows in bulk, not one by one.

`FromIter` reads the lines lazily. `MapFilter` parses each line and skips bad rows. `Split` groups rows into batches for one insert each.

::: code-group

<div class="vp-block active" data-title="Diagram"><img src="../guide/diagrams/example-file-import.svg" alt="MapFilter parses lines and skips x; Split groups the rows into insert batches"></div>

```go [Code]
type Row struct {
    ID   int
    Name string
}

func importFile(ctx context.Context, r io.Reader, db *sql.DB) error {
    scanner := bufio.NewScanner(r)
    lines := func(yield func(string) bool) {
        for scanner.Scan() {
            if !yield(scanner.Text()) {
                return
            }
        }
    }

    rows := goflow.MapFilter(goflow.FromIter(ctx, lines), func(ctx context.Context, line string) (Row, bool, error) {
        id, name, ok := strings.Cut(line, ",")
        if !ok {
            return Row{}, false, nil // skip a malformed line
        }
        n, err := strconv.Atoi(id)
        if err != nil {
            return Row{}, false, nil // skip a row with a bad ID
        }
        return Row{ID: n, Name: name}, true, nil
    })

    return goflow.Split(rows, 500).ForEach(func(ctx context.Context, batch []Row) error {
        return insertRows(ctx, db, batch) // one INSERT for up to 500 rows
    })
}
```

:::

The file is never held in memory. Only one batch of 500 rows is in memory at a time.

## Error Handling

An import can fail in 3 places:

1. The reader fails, for example on a broken network file. `FromIter` cannot report this error, so check `scanner.Err()` after the stream.
2. A line is not valid. Log it with its line number. Stop the import when there are too many bad lines.
3. An insert fails. `ForEach` returns the first error. Cancel the context, so that the reader and the parser stop too.

```go
type line struct {
    n    int
    text string
}

func importFileChecked(parent context.Context, r io.Reader, db *sql.DB) error {
    ctx, cancel := context.WithCancel(parent)
    defer cancel() // stops the reader and the parser when an insert fails

    scanner := bufio.NewScanner(r)
    lines := func(yield func(line) bool) {
        n := 0
        for scanner.Scan() {
            n++
            if !yield(line{n: n, text: scanner.Text()}) {
                return
            }
        }
    }

    skipped := 0 // written only by the MapFilter stage
    rows := goflow.MapFilter(goflow.FromIter(ctx, lines), func(ctx context.Context, l line) (Row, bool, error) {
        row, err := parseRow(l.text)
        if err != nil {
            skipped++
            slog.WarnContext(ctx, "skip line", "line", l.n, "error", err)
            return Row{}, false, nil
        }
        return row, true, nil
    })

    err := goflow.Split(rows, 500).ForEach(func(ctx context.Context, batch []Row) error {
        if err := insertRows(ctx, db, batch); err != nil {
            return fmt.Errorf("insert rows %d to %d: %w", batch[0].ID, batch[len(batch)-1].ID, err)
        }
        return nil
    })
    if err != nil {
        return err
    }
    if err := ctx.Err(); err != nil {
        return err // the import did not finish
    }

    // The stream is fully read here, so scanner and skipped are safe to read.
    if err := scanner.Err(); err != nil {
        return fmt.Errorf("read file: %w", err)
    }
    if skipped > 100 {
        return fmt.Errorf("import: %d bad lines", skipped)
    }
    return nil
}
```

::: warning
Each batch commits on its own. When batch 3 fails, batches 1 and 2 stay in the database. Make the insert idempotent, for example with an upsert, so that you can run the import again.
:::
