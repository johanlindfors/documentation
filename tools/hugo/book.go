package hugo

import (
  "context"
  "github.com/peter-mount/documentation/tools/util"
  "os"
  "path/filepath"
  "strings"
  "time"
)

type BookHandler func(context.Context, *Book) error

func WithBook() BookHandler {
  return nil
}

func (a BookHandler) Then(b BookHandler) BookHandler {
  if a == nil {
    return b
  }
  return func(ctx context.Context, book *Book) error {
    if err := a(ctx, book); err != nil {
      return err
    }
    return b(ctx, book)
  }
}

func (a BookHandler) Do(ctx context.Context, book *Book) error {
  if a == nil {
    return nil
  }
  return a(ctx, book)
}

func (b *Book) IsExcelPresent() bool {
  return b.excel != nil
}

func (a BookHandler) IfExcelPresent(h func(context.Context, *Book, util.ExcelBuilder) error) BookHandler {
  return a.Then(func(ctx context.Context, b *Book) error {
    if b.IsExcelPresent() {
      return h(context.WithValue(ctx, "excel", b.excel), b, b.excel)
    }
    return nil
  })
}

type BookGeneratorHandler func(context.Context, *Book, string) error

func WithBookGenerator() BookGeneratorHandler {
  return nil
}

func (a BookGeneratorHandler) Do(ctx context.Context, book *Book, s string) error {
  if a != nil {
    return a(ctx, book, s)
  }
  return nil
}

func (a BookGeneratorHandler) Then(b BookGeneratorHandler) BookGeneratorHandler {
  if a == nil {
    return b
  }
  return func(ctx context.Context, book *Book, n string) error {
    err := a(ctx, book, n)
    if err != nil {
      return err
    }
    return b(ctx, book, n)
  }
}

func (a BookHandler) ForEachGenerator(f BookGeneratorHandler) BookHandler {
  return a.Then(func(ctx context.Context, book *Book) error {
    return book.Generate.ForEach(func(s string) error {
      return f(context.WithValue(ctx, "bookGeneratorHandler", s), book, s)
    })
  })
}

type Books []*Book

func (bs Books) ForEach(ctx context.Context, f BookHandler) error {
  for _, b := range bs {
    err := f(ctx, b)
    if err != nil {
      return err
    }
  }
  return nil
}

// Book defines a book that's rendered as pdf
type Book struct {
  BookCopyright                   // Copyright of book
  ID            string            `yaml:"id"` // ID of the book, e.g. "bbc" or "6502"
  FrontImage    BookCopyright     `yaml:"frontImage"` // Copyright of front image
  PDF           PDF               `yaml:"pdf"` // Custom PDF config for just this book
  Generate      util.StringSlice  `yaml:"generate"` // List of generators to run on this book
  modified      time.Time         `yaml:"-"` // Last Modified time
  excel         util.ExcelBuilder `yaml:"-"` // Excel builder if present
  excelWritten  bool              // Set to true if the file has been written
}

type BookCopyright struct {
  Title     string `yaml:"title"`     // Title of book, default title from main page
  SubTitle  string `yaml:"subTitle"`  // SubTitle
  Author    string `yaml:"author"`    // Author of book, default ""
  SubAuthor string `yaml:"subAuthor"` // SubAuthor of book, default ""
  Copyright string `yaml:"copyright"` // Copyright
}

func (b *Book) ExcelRunOnce(f func() error) func() error {
  return func() error {
    if !b.excelWritten {
      b.excelWritten = true
      return f()
    }
    return nil
  }
}

func (b *Book) ContentPath() string {
  return "content/" + b.ID + "/"
}

func (b *Book) Modified() time.Time {
  if b.modified.IsZero() {
    _ = filepath.Walk(b.ContentPath(), func(_ string, info os.FileInfo, _ error) error {
      if !info.IsDir() && info.ModTime().After(b.modified) {
        b.modified = info.ModTime()
      }
      return nil
    })
  }
  return b.modified
}

func replace(s, f string, h func() string) string {
  if strings.Contains(s, f) {
    return strings.ReplaceAll(s, f, h())
  }
  return s
}

func (b *Book) Expand(s string) string {
  // Modified date of content
  s = replace(s, "${modified}", func() string {
    return b.Modified().Format(time.RFC1123)
  })

  // Expand book title or use default from chrome
  s = replace(s, "${title}", func() string {
    t := b.Title
    if t == "" {
      t = "<span class='title'></span>"
    }
    return strings.ReplaceAll(s, "${title}", t)
  })

  s = replace(s, "${author}", func() string {
    return b.Author
  })

  s = replace(s, "${copyright}", func() string {
    return b.Copyright
  })

  return s
}

func (b *Book) GetExcel() util.ExcelBuilder {
  if b.excel == nil {
    b.excel = util.NewExcelBuilder()
  }
  return b.excel
}

func (b *Book) SetExcel(eb util.ExcelBuilder) {
  b.excel = eb
}

// Do runs a function against this instance. When it exits it removes any resources the Book has used freeing up memory.
func (b *Book) Do(f func(*Book) error) error {
  defer func() {
    b.excel = nil
  }()
  return f(b)
}
