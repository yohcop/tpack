package main

import (
  "fmt"
  "flag"
  "image"
  "image/color"
  "image/png"
  "io/ioutil"
  "text/template"
  "path/filepath"
  "sort"
  "strings"
  "os"
)

var dir = flag.String("d", ".", "Directory to look for images")
var padding = flag.Int("p", 0, "Padding in pixels")
var size = flag.Int("s", 512, "Output texture size")
var out = flag.String("o", "output.png", "Output png")
var tpl = flag.String("t", "tpack.tpl", "Output template")
var cfg = flag.String("c", "output.cfg", "Output config")
var verbose = flag.Bool("v", false, "Verbose")

type Rect struct {
  W, H int
  X, Y int
  Name string
  // The above name, but transformed into a usually valid ID,
  // mostly - and . are replaced by _.
  NameId string
  Img image.Image
}

func (r *Rect) FitsIn(that *Rect) bool {
  return r.W < that.W && r.H < that.H
}

func (r *Rect) Overlaps(that *Rect) bool {
  r1Left := r.X
  r1Right := r.X + r.W
  r1Top := r.Y
  r1Bottom := r.Y + r.H

  r2Left := that.X
  r2Right := that.X + that.W
  r2Top := that.Y
  r2Bottom := that.Y + that.H

  return r1Left < r2Right &&
         r1Right > r2Left &&
         r1Top < r2Bottom &&
         r1Bottom > r2Top
}

type Rects []*Rect
func (r Rects) Len() int {
  return len(r)
}
func (r Rects) Swap(i, j int) {
  r[i], r[j] = r[j], r[i]
}
func (r Rects) Less(i, j int) bool {
  return r[i].W * r[i].H > r[j].W * r[j].H
}
func (r Rects) ColorModel() color.Model {
  return color.RGBAModel
}
func (r Rects) Bounds() image.Rectangle {
  return image.Rect(0, 0, *size, *size)
}
func (r Rects) At(x, y int) color.Color {
  for _, rect := range r {
    if rect.X <= x && rect.X + rect.W > x &&
        rect.Y <= y && rect.Y + rect.H > y {
      return rect.Img.At(x - rect.X, y - rect.Y)
    }
  }
  return color.RGBA{0, 0, 0, 0}
}

type Point struct {
  x, y int
}

func Fits(r *Rect, into *Rect, at *Point, placed Rects) bool {
  if into.W < at.x + r.W || into.H < at.y + r.H {
    return false
  }
  r.X = at.x
  r.Y = at.y
  for _, p := range placed {
    if r.Overlaps(p) {
      return false
    }
  }
  return true
}

func Pack(rects Rects, into *Rect, pts []*Point) {
  if len(rects) == 0 {
    return
  }
  for i, r := range rects {
    placed := false
    for n, p := range pts {
      if Fits(r, into, p, rects[0:i]) {
        pts = append(pts[0:n], pts[n+1:]...)
        pts = append(pts, &Point{p.x + r.W, p.y})
        pts = append(pts, &Point{p.x, p.y + r.H})
        placed = true
        break;
      }
    }
    if !placed {
      fmt.Printf("Could not make rectangle %v fit :(\n", r)
      return
    }
  }
}

func makeIdName(name string) string {
  s := strings.Split(name, ".")
  if len(s) > 1 {
    name = strings.Join(s[0:len(s) - 1], "_")
  }
  return strings.Replace(
      strings.Replace(name, "-", "_", -1),
      ".", "_", -1)
}

func ReadImage(path string, filename string) *Rect {
  in, err := os.Open(filepath.Join(path, filename))
  defer in.Close()
  if err != nil {
    fmt.Println(err.Error())
    return nil
  }
  img, _, err := image.Decode(in)
  if err != nil {
    fmt.Println(err.Error())
    return nil
  }
  r := &Rect{
      W: img.Bounds().Max.X + *padding,
      H: img.Bounds().Max.Y + *padding,
      Name: filename,
      NameId: makeIdName(filename),
      Img: img,
  }
  if *verbose {
    fmt.Printf("Image %s: %dx%d\n", filename, r.W, r.H)
  }
  return r
}

func LoadImages(dir string) Rects {
  r := Rects{}

  d, err := os.Open(dir)
  defer d.Close()
  if err != nil {
    fmt.Println(err.Error())
    return r
  }
  files, err := d.Readdir(-1)
  if err != nil {
    fmt.Println(err.Error())
    return r
  }

  for _, f := range files {
    if !f.IsDir() {
      r = append(r, ReadImage(dir, f.Name()))
    }
  }
  return r
}

func WriteFinalSprite(out string, rects Rects) {
  f, err := os.OpenFile(out, os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0666);
  defer f.Close()
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  png.Encode(f, rects)
}

func WriteSpriteConfig(out string, rects Rects, into *Rect, tpl string) {
  str, err := ioutil.ReadFile(tpl)
  t := template.Must(template.New("tpl").Parse(string(str)))

  f, err := os.OpenFile(out, os.O_WRONLY | os.O_CREATE, 0666);
  defer f.Close()
  if err != nil {
    fmt.Println(err.Error())
    return
  }
  f.Truncate(0)

  m := map[string]interface{}{
     "img": into,
     "rects": rects,
  }
  t.Execute(f, m)
}

func main() {
  flag.Parse()
  rects := LoadImages(*dir)
  sort.Sort(rects)
  into := &Rect{W: *size, H: *size}
  Pack(rects, into, []*Point{&Point{0, 0}})
  WriteFinalSprite(*out, rects)
  WriteSpriteConfig(*cfg, rects, into, *tpl)
}
