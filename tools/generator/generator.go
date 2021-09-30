package generator

import (
  "context"
  "fmt"
  "github.com/peter-mount/documentation/tools/hugo"
  "github.com/peter-mount/documentation/tools/util"
  "github.com/peter-mount/go-kernel"
  "log"
)

// Handler performs an action against a Book.
// Handler's can either do all of their work in one go or pass Task's to the Generator to be run after
// all other Handler's have run.
type Handler func(*hugo.Book) error

// Then returns a GeneratorHandler that calls this one then if no error the next one forming a chain
func (h Handler) Then(next Handler) Handler {
  return func(b *hugo.Book) error {
    err := h(b)
    if err != nil {
      return err
    }
    return next(b)
  }
}

// HandlerOf returns a Handler that will invoke each supplied GeneratorHandler in a single instance.
// This is a convenience form of calling Then() on each one.
func HandlerOf(handlers ...Handler) Handler {
  switch len(handlers) {
  case 0:
    return func(_ *hugo.Book) error {
      return nil
    }
  case 1:
    return handlers[0]
  default:
    a := handlers[0]
    for _, b := range handlers[1:] {
      a = a.Then(b)
    }
    return a
  }
}

// RunOnce will call a Handler once. It uses a pointer to a boolean to store this state.
// It's useful for simple tasks but should be treated as Deprecated as it only works for one Book not multiple books.
func (h Handler) RunOnce(s *bool, b Handler) Handler {
  return h.Then(func(book *hugo.Book) error {
    if !*s {
      *s = true
      return b(book)
    }
    return nil
  })
}

// Task is a task that the Generator must run once all other Handler's have been run.
// They are usually tasks created by those Handlers.
type Task func() error

// Generator is a kernel Service which handles the generation of content based on page metadata.
type Generator struct {
  config     *hugo.Config // Configuration
  bookShelf  *hugo.BookShelf
  generators map[string]Handler // Map of available generators
  tasks      util.PriorityQueue
}

func (g *Generator) Name() string {
  return "generator"
}

func (g *Generator) Init(k *kernel.Kernel) error {

  service, err := k.AddService(&hugo.Config{})
  if err != nil {
    return err
  }
  g.config = service.(*hugo.Config)

  service, err = k.AddService(&hugo.BookShelf{})
  if err != nil {
    return err
  }
  g.bookShelf = service.(*hugo.BookShelf)

  return nil
}

func (g *Generator) Start() error {
  g.generators = make(map[string]Handler)
  return nil
}

// Register a named Handler
func (g *Generator) Register(n string, h Handler) *Generator {
  if _, exists := g.generators[n]; exists {
    panic(fmt.Errorf("GeneratorHandler %s already registered", n))
  }

  g.generators[n] = h
  return g
}

// AddTask appends a Task to be performed once all Handler's have run.
func (g *Generator) AddTask(t Task) *Generator {
  g.tasks = g.tasks.Add(t)
  return g
}

// AddPriorityTask appends a Task to be performed once all Handler's have run.
func (g *Generator) AddPriorityTask(p int, t Task) *Generator {
  g.tasks = g.tasks.AddPriority(p, t)
  return g
}

func (g *Generator) Run() error {
  if err := g.bookShelf.Books().ForEach(
    context.Background(),
    hugo.WithBook().
        ForEachGenerator(
          hugo.WithBookGenerator().
            Then(g.invokeGenerator)).
        IfExcelPresent(func(ctx context.Context, book *hugo.Book, excel util.ExcelBuilder) error {
          // Now append this as a task, priority 100 so that it runs after most tasks
          g.AddPriorityTask(100, book.ExcelRunOnce(func() error {
            return excel.FileHandler().
              Write(util.ReferenceFilename(book.ContentPath(), "", "reference.xlsx"), book.Modified())
          }))
          return nil
        })); err != nil {
    return err
  }

  return g.tasks.Drain(func(i interface{}) error {
    return i.(Task)()
  })
}

func (g *Generator) invokeGenerator(ctx context.Context, book *hugo.Book, n string) error {
  h, exists := g.generators[n]
  if exists {
    return h(book)
  }

  // Log a warning but ignore - could be an invalid config or the generator is not deployed.
  // Originally this was a fatal error, but now we just ignore to allow custom tools to be run
  log.Printf("book %s GeneratorHandler %s is not registered", book.ID, n)
  return nil
}
