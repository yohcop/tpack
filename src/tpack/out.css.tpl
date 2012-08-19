{{range .rects}}{{if .}}/* {{.Name}} */
.{{.NameId}} {
  left: {{.X}}px;
  right: {{.Y}}px;
  width: {{.W}}px;
  height: {{.H}}px
}{{end}}
{{end}}
