package m68k

import (
  "github.com/peter-mount/documentation/tools/hugo"
  "github.com/peter-mount/documentation/tools/util"
  util2 "github.com/peter-mount/go-kernel/util"
  "github.com/peter-mount/go-kernel/util/strings"
)

type IndexGenerator struct {
  Prefix    string
  Name      string
  Title     string
  Desc      string
  Class     string
  Paginator func(int, interface{}) bool
  Header    func(strings.StringSlice, int) strings.StringSlice
  Body      func(strings.StringSlice, int, interface{}) strings.StringSlice
}

func (i *IndexGenerator) WriteFile(book *hugo.Book, iterator util2.Iterator) error {
  return util.ReferenceFileBuilder(
    i.Title,
    i.Desc,
    "manual",
    10,
    book.Modified(),
  ).
    //Then(inst.writeOpCodes(prefix, inst.opCodes)).
    WrapAsFrontMatter().
      Then(func(slice strings.StringSlice) (strings.StringSlice, error) {
        rowCount := 0

        slice = i.startPage(rowCount, slice)
        for iterator.HasNext() {
          row := iterator.Next()
          if rowCount > 0 && i.Paginator != nil && i.Paginator(rowCount, row) {
            slice = i.endPage(slice)
            slice = i.startPage(rowCount, slice)
          }

          slice = i.Body(slice, rowCount, row)
          rowCount++
        }
        slice = i.endPage(slice)
        return slice, nil
      }).
    FileHandler().
    Write(util.ReferenceFilename(book.ContentPath(), i.Name, "_index.html"), book.Modified())
}

func (i *IndexGenerator) startPage(rowCount int, slice strings.StringSlice) strings.StringSlice {
  slice = append(slice, "<div class='"+i.Class+"'><table>")
  if i.Header != nil {
    slice = i.Header(slice, rowCount)
  }
  return slice
}

func (i *IndexGenerator) endPage(slice strings.StringSlice) strings.StringSlice {
  slice = append(slice, "</tbody></table></div>")
  return slice
}
