package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	dataPath string = "data"
	tmplPath string = "tmpl"
)

var validLink = regexp.MustCompile(`\[[a-zA-Z0-9]+\]`)

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) HTMLBody() template.HTML {
	escaped := template.HTMLEscapeString(string(p.Body))
	linked := validLink.ReplaceAllStringFunc(escaped, func(match string) string {
		title := match[1 : len(match)-1]
		return fmt.Sprintf(`<a href="/view/%s">%s</a>`, title, title)
	})
	linkedWithNewlines := strings.ReplaceAll(linked, "\n", "<br>")

	return template.HTML(linkedWithNewlines)
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	filePath := filepath.Join(dataPath, filename)
	err := os.MkdirAll(dataPath, 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	filePath := filepath.Join(dataPath, filename)

	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

var tmplEdit = filepath.Join(tmplPath, "edit.html")
var tmplView = filepath.Join(tmplPath, "view.html")
var templates = template.Must(template.ParseFiles(tmplEdit, tmplView))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/view/", makeHandler(viewHandler))
	mux.HandleFunc("/edit/", makeHandler(editHandler))
	mux.HandleFunc("/save/", makeHandler(saveHandler))

	port := ":8080"
	fmt.Printf("Starting server at http://localhost%v\n", port)
	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
