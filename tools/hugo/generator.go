package hugo

import (
  "fmt"
  "github.com/peter-mount/go-kernel"
)

// GeneratorHandler performs an action against a Book
type GeneratorHandler func(*Book) error

// Then returns a GeneratorHandler that calls this one then if no error the next one forming a chain
func (h GeneratorHandler) Then(next GeneratorHandler) GeneratorHandler {
  return func(b *Book) error {
    err := h(b)
    if err == nil {
      err = next(b)
    }
    return err
  }
}

type Generator struct {
  config     *Config                     // Configuration
  generators map[string]GeneratorHandler // Map of available generators
}

func (g *Generator) Name() string {
  return "generator"
}

func (g *Generator) Init(k *kernel.Kernel) error {

  service, err := k.AddService(&Config{})
  if err != nil {
    return err
  }
  g.config = service.(*Config)

  g.generators = make(map[string]GeneratorHandler)
  return nil
}

func (g *Generator) register(n string, h GeneratorHandler) *Generator {
  if _, exists := g.generators[n]; exists {
    panic(fmt.Errorf("GeneratorHandler %s already registered", n))
  }

  g.generators[n] = h
  return g
}

// Register creates a named GeneratorHandler composed of the supplied handlers
func (g *Generator) Register(n string, handlers ...GeneratorHandler) *Generator {
  switch len(handlers) {
  case 0:
    panic(fmt.Errorf("no GeneratorHandlers defined for %s", n))
  case 1:
    return g.register(n, handlers[0])
  default:
    h := handlers[0]
    for _, next := range handlers[1:] {
      h = h.Then(next)
    }
    return g.register(n, h)
  }
}

func (g *Generator) Run() error {
  return g.config.Books.ForEach(func(book *Book) error {
    return book.Generate.ForEach(func(n string) error {
      h, exists := g.generators[n]
      if !exists {
        return fmt.Errorf("book %s GeneratorHandler %s is not registered", book.ID, n)
      }

      return h(book)
    })
  })
}
