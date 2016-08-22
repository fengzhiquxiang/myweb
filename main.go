package main

import (
	"fmt"
	"log"
	"net/http"
	"html/template"

	"database/sql"
    _ "github.com/go-sql-driver/mysql"

    "encoding/json"
    "net/url"
    "io/ioutil"
    "encoding/xml"
)

type Book struct{
	Title string
	Author string
	MostPopular string
	ID string
}

type Page struct {
	Books []Book
}

type SearchResult struct{
	Title string `xml:"title,attr"`
	Author string `xml:"author,attr"`
	Year string `xml:"hyr,attr"`
	ID string `xml:"owi,attr"`
}

func main() {
	templates := template.Must(template.ParseFiles("src/myweb/templates/index.html"))

	db, err := sql.Open("mysql", "root:luzlhefh@/mywebdb")
	defer db.Close()
	if err != nil {
		fmt.Println("db is available")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		if err := db.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		rs, err := db.Query("select * from foo")
		if err != nil {
			// http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err)
		}
		var p Page
		for rs.Next() {
			var b Book
			if err := rs.Scan(&b.Title, &b.Author, &b.MostPopular, &b.ID); err != nil{
				log.Fatal(err)
			}
			p.Books = append(p.Books, b)
		}
		if err := templates.ExecuteTemplate(w, "index.html", p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
    })

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request){
		var results []SearchResult
		var err error
		query := r.FormValue("search")
		if results, err = search(query); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(results); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
    })

	http.HandleFunc("/books/add", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var resp *http.Response

		if resp, err =http.Get("http://classify.oclc.org/classify2/Classify?summary=true&owi=" + url.QueryEscape(r.FormValue("id"))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		defer resp.Body.Close()
		var body []byte
		if body, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Println("resp body err")
		}

		var rs ClassifyBookResponse
		xml.Unmarshal(body,&rs)

		if err = db.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		_, err = db.Exec("insert into foo (title, author, mostpopular, id) value(?,?,?,?)", rs.BookData.Title, rs.BookData.Author, rs.Classification.MostPopular, rs.BookData.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		if err = json.NewEncoder(w).Encode(rs); err != nil {
			log.Fatal(err)
		}
	})

	http.HandleFunc("/books/delete", func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err = db.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		_, err = db.Exec("delete from foo where ID=?", url.QueryEscape((r.FormValue("id"))[5:]))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	fmt.Println(http.ListenAndServe(":8000", nil))
}

type ClassifySearchResponse	struct{
	Results []SearchResult `xml:"works>work"`
}

type BookSelectResponse struct{
	BookData struct{
		Title string `xml:"title,attr"`
		Author string `xml:"author,attr"`
		ID string `xml:"owi,attr"`
	} `xml:"work"`
	AuthorData struct {
		Author []string `xml:"author"`
	} `xml:"authors>author"`
}

type ClassifyBookResponse struct{
	BookData struct{
		Title string `xml:"title,attr"`
		Author string `xml:"author,attr"`
		ID string `xml:"owi,attr"`
	} `xml:"work"`
	Classification struct{
		MostPopular string `xml:"sfa,attr"`
	} `xml:"recommendations>ddc>mostPopular"`
}

func search(query string) ([]SearchResult, error) {
	var resp *http.Response
	var err error

	if resp, err =http.Get("http://classify.oclc.org/classify2/Classify?summary=true&title=" + url.QueryEscape(query)); err != nil {
		return []SearchResult{}, err
	}

	defer resp.Body.Close()
	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return []SearchResult{}, err 
	}

	var c ClassifySearchResponse
	err = xml.Unmarshal(body,&c)
	return c.Results, err 
}