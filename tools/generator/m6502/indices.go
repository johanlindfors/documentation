package m6502

import (
  "context"
  "fmt"
  "github.com/peter-mount/documentation/tools/generator"
  "github.com/peter-mount/documentation/tools/hugo"
  "github.com/peter-mount/documentation/tools/util"
  "github.com/peter-mount/documentation/tools/util/task"
  "log"
  "sort"
  "strconv"
  "strings"
)

func delayOpTask(t task.Task) task.Task {
  return func(ctx context.Context) error {
    task.GetQueue(ctx).AddPriorityTask(25,
      task.Of(t).
        WithContext(ctx, generator.BookKey))
    return nil
  }
}

func (s *M6502) writeOpsIndex(ctx context.Context) error {
  _ = s.autodoc.GetApi(ctx)

  book := generator.GetBook(ctx)
  inst := s.Instructions(book)

  sort.SliceStable(inst.opCodes, func(i, j int) bool {
    a := inst.opCodes[i].Code
    b := inst.opCodes[j].Code

    if ai, err1 := strconv.ParseInt(strings.ReplaceAll(a, "nn", ""), 16, 64); err1 == nil {
      if bi, err2 := strconv.ParseInt(strings.ReplaceAll(b, "nn", ""), 16, 64); err2 == nil {
        return ai < bi
      } else {
        log.Println("b", b, err2)
      }
    } else {
      log.Println("a", a, err1)
    }

    return a<b
  })

  return s.writeFile(
    book,
    inst,
    "codes",
    "opcodes",
    "Instruction List by opcode",
    "6502 instructions by hex opcode",
  )
}

func (s *M6502) writeOpsHexIndex(ctx context.Context) error {
  _ = s.autodoc.GetApi(ctx)

  book := generator.GetBook(ctx)
  inst := s.Instructions(book)

  sort.SliceStable(inst.opCodes, func(i, j int) bool {
    if inst.opCodes[i].Op == inst.opCodes[j].Op {
      return inst.opCodes[i].Addressing < inst.opCodes[j].Addressing
    }
    return inst.opCodes[i].Op < inst.opCodes[j].Op
  })

  return s.writeFile(
    book,
    inst,
    "codes",
    "instructions",
    "Instruction List by name",
    "6502 instructions by name",
  )
}

func (s *M6502) writeOpsHexGrid(ctx context.Context) error {
  _ = s.autodoc.GetApi(ctx)

  book := generator.GetBook(ctx)
  inst := s.Instructions(book)

  return util.ReferenceFileBuilder("Opcode Matrix", "Instructions shown in an Opcode Matrix", "manual", 10).
      Then(NewHexGrid().
        Opcode(inst.opCodes...).
        FileBuilder()).
    WrapAsFrontMatter().
    FileHandler().
    Write(util.ReferenceFilename(book.ContentPath(), "hexgrid", "_index.html"), book.Modified())
}

func (s *M6502) writeFile(book *hugo.Book, inst *Instructions, prefix, name, title, desc string) error {
  err := util.ReferenceFileBuilder(
    title,
    desc,
    "manual",
    10,
  ).
    //Then(inst.writeOpCodes(prefix, inst.opCodes)).
    WrapAsFrontMatter().
      Then(func(slice util.StringSlice) (util.StringSlice, error) {
        slice = append(slice, "<div class='opIndex'>", "<table>")
        for _, op := range inst.opCodes {
          slice = append(slice, fmt.Sprintf(
            "<tr><td>%s</td><td>%s</td></tr>",
            op.Op,
            op.Code,
          ))
        }
        slice = append(slice, "</table>", "</div>")
        return slice, nil
      }).
    FileHandler().
    Write(util.ReferenceFilename(book.ContentPath(), name, "_index.html"), book.Modified())
  if err != nil {
    return err
  }

  // Use a copy as we reorder s.opCodes so when excel is processed later it gets just the last ordering
  codes := append([]*Opcode{}, inst.opCodes...)

  return util.WithTable().
    AsCSV(book.StaticPath(name+".csv"), book.Modified()).
    AsExcel(s.excel.Get(book.ID, book.Modified())).
      Do(&util.Table{
        Title: name,
        Columns: []string{
          "Decimal",
          "Hex",
          "Instruction",
          "Addressing",
          "Len(byte)",
          "Cycles",
        },
        RowCount: len(codes),
        GetRow: func(r int) interface{} {
          return codes[r]
        },
        Transform: func(i interface{}) []interface{} {
          o := i.(*Opcode)
          return []interface{}{
            o.Order,
            o.Code,
            o.Op,
            o.Addressing,
            o.Bytes.Value,
            o.Cycles.Value,
          }
        },
      })
}
