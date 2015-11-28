package main

import (
	"os"
	"log"
	"io/ioutil"
	"path/filepath"
	"html/template"
	"strconv"
	"net/http"
	"sort"
)

// directory struct: {root}/{book title}/{chapter num}/{block num}
// when program begin, it will look through the dirs
// and make following structs.

type book struct {
	Title string
	Chapters []chapter
}

type chapter struct {
	Num int
	Snippets []snippet
}

// snippet is a string block of resonable length.
// user will decide how long will it be.
type snippet struct {
	Orig string
	Trans string
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatal(err)
	}
	books := scanRootDir()
	err = t.Execute(w, books[0])
	if err != nil {
		log.Fatal(err)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	// Why r.Form["x"] get in form with []string ?
	typ := r.Form["type"][0]
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	switch typ {
	case "save":
		orig := r.Form["orig"][0]
		trans := r.Form["trans"][0]
		saveSnippet(path, orig, trans)
	case "new":
		orig := r.Form["orig"][0]
		trans := r.Form["trans"][0]
		newSnippet(path, orig, trans)
	case "remove":
		removeSnippet(path)
	}
}

func saveSnippet(path, orig, trans string) {
	err := ioutil.WriteFile(path + "/orig", []byte(orig), 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path + "/trans", []byte(trans), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func newSnippet(path, orig, trans string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
	saveSnippet(path, orig, trans)
}

func removeSnippet(path string) {
	err := os.Remove(filepath.Join(path, "orig"));
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(filepath.Join(path, "trans"));
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(path)
	if err != nil {
		log.Fatal(err)
	}
	// after remove the snippet dir, the later dirs should renamed to fill hole.
	// if 2 is removed. 3 -> 2, 4 -> 3
	chapd, snipd := filepath.Split(path)
	rmIndex, err := strconv.Atoi(snipd)
	if err != nil {
		log.Fatal(err)
	}
	files, err := ioutil.ReadDir(chapd)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(byIndex(files))
	for _, f := range files {
		if !f.IsDir() {
			log.Fatal("chapter directory should only have snippet directories. file found")
		}
		i, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Fatal(err)
		}
		if i <= rmIndex {
			continue
		}
		newName := strconv.Itoa(i-1)
		err = os.Rename(filepath.Join(chapd, f.Name()), filepath.Join(chapd, newName))
		if err != nil {
			log.Fatal(err)
		}
	}
}

type byIndex []os.FileInfo

func (b byIndex) Len() int {
	return len(b)
}

func (b byIndex) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byIndex) Less(i, j int) bool {
	return atoi(b[i].Name()) < atoi(b[j].Name())
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

// scanRootDir scan and return books in root dir with alphabetical order.
func scanRootDir() []book {
	root := "./testroot"
	bookFiles, err := ioutil.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}
	books := make([]book, 0)
	for _, bf := range bookFiles {
		if !bf.IsDir() {
			continue
		}
		bk := book{Title:bf.Name(), Chapters:make([]chapter, 0)}
		bdir := filepath.Join(root, bf.Name())
		chapterFiles, err := ioutil.ReadDir(bdir)
		if err != nil {
			log.Fatal(err)
		}
		for _, cf := range chapterFiles {
			if !cf.IsDir() {
				continue
			}
			cnum, err := strconv.Atoi(cf.Name())
			if err != nil {
				log.Fatal(err)
			}
			chap := chapter{Num:cnum, Snippets:make([]snippet, 0)}
			cdir := filepath.Join(bdir, cf.Name())
			snippetFiles, err := ioutil.ReadDir(cdir)
			if err != nil {
				log.Fatal(err)
			}
			var orig, trans string
			for _, f := range snippetFiles {
				if !f.IsDir() {
					continue
				}
				sdir := filepath.Join(cdir, f.Name())
				files, err := ioutil.ReadDir(sdir)
				if err != nil {
					log.Fatal(err)
				}
				for _, f := range files {
					// lasts are files
					if f.IsDir() {
						continue
					}
					fpath := filepath.Join(sdir, f.Name())
					if f.Name() == "orig" {
						o, err := ioutil.ReadFile(fpath)
						if err != nil {
							log.Fatal(err)
						}
						orig = string(o)
					} else if f.Name() == "trans" {
						t, err := ioutil.ReadFile(fpath)
						if err != nil {
							log.Fatal(err)
						}
						trans = string(t)
					}
				}
				snip := snippet{Orig:orig, Trans:trans}
				chap.Snippets = append(chap.Snippets, snip)
			}
			bk.Chapters = append(bk.Chapters, chap)
		}
		books = append(books, bk)
	}
	return books
}

func main() {
	rootpath := "testroot"

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		updateHandler(w, r, rootpath)
	})

	fs := http.StripPrefix("/script/", http.FileServer(http.Dir("script/")))
	http.Handle("/script/", fs)

	fs = http.StripPrefix("/css/", http.FileServer(http.Dir("css/")))
	http.Handle("/css/", fs)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rootHandler(w, r)
	})
	http.ListenAndServe(":8080", nil)
}
