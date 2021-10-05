package pdf

import (
  "context"
  "flag"
  "github.com/chromedp/cdproto/page"
  "github.com/chromedp/chromedp"
  "github.com/peter-mount/documentation/tools/hugo"
  "github.com/peter-mount/documentation/tools/util"
  "github.com/peter-mount/go-kernel"
  "log"
)

// PDF tool that handles the generation of PDF documentation of a "book"
type PDF struct {
  config    *hugo.Config // Config
  bookShelf *hugo.BookShelf
  chromium  *hugo.Chromium // Chromium browser
  enable    *bool          // Is PDF generation enabled
}

func (p *PDF) Name() string {
  return "PDF"
}

func (p *PDF) Init(k *kernel.Kernel) error {
  p.enable = flag.Bool("p", false, "disable pdf generation")

  service, err := k.AddService(&hugo.Config{})
  if err != nil {
    return err
  }
  p.config = service.(*hugo.Config)

  service, err = k.AddService(&hugo.Chromium{})
  if err != nil {
    return err
  }
  p.chromium = service.(*hugo.Chromium)

  service, err = k.AddService(&hugo.BookShelf{})
  if err != nil {
    return err
  }
  p.bookShelf = service.(*hugo.BookShelf)

  // We need a webserver & must run after hugo
  return k.DependsOn(&hugo.Webserver{}, &hugo.Hugo{})
}

// Run through args for book id's and generate the PDF's
func (p *PDF) Run() error {
  if *p.enable {
    return nil
  }

  return p.bookShelf.Books().ForEach(context.Background(), p.generate)
}

func (p *PDF) generate(ctx context.Context, book *hugo.Book) error {
  log.Println("Generating PDF for", book.ID)
  return p.chromium.Run(p.printToPDF(book))
}

// print a specific pdf page.
func (p *PDF) printToPDF(book *hugo.Book) chromedp.Tasks {
  url := p.config.WebPath("%s/_print/", book.ID)

  pdf := book.PDF

  return chromedp.Tasks{
    chromedp.Navigate(url),
    chromedp.ActionFunc(func(ctx context.Context) error {
      buf, _, err := page.PrintToPDF().
        WithPrintBackground(pdf.PrintBackground).
        WithMarginTop(pdf.Margin.Top).
        WithMarginBottom(pdf.Margin.Bottom).
        WithMarginLeft(pdf.Margin.Left).
        WithMarginRight(pdf.Margin.Right).
        WithLandscape(pdf.Landscape).
        WithPaperWidth(pdf.Width).
        WithPaperHeight(pdf.Height).
        WithDisplayHeaderFooter(!pdf.DisableHeaderFooter).
        WithHeaderTemplate(book.Expand(pdf.Header)).
        WithFooterTemplate(book.Expand(pdf.Footer)).
        Do(ctx)

      if err != nil {
        return err
      }

      return util.ByteFileHandler(buf).
        Write("static/static/book/"+book.ID+".pdf", book.Modified())
    }),
  }
}
