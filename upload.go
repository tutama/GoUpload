package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"database/sql"

	_ "github.com/lib/pq"
)

func main() {
	http.HandleFunc("/", routeIndexGet)
	http.HandleFunc("/process", routeSubmitPost)

	fmt.Println("server started at localhost:9000")
	http.ListenAndServe(":9000", nil)
}

func routeIndexGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	os.Setenv("myAuth", "FooBar")
	myAuthEnvValue := os.Getenv("myAuth")
	myAuth := map[string]interface{}{"myAuth": myAuthEnvValue}

	var tmpl = template.Must(template.ParseFiles("view.html"))
	var err = tmpl.Execute(w, myAuth)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeSubmitPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(1024); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check auth
	myAuth := r.FormValue("myAuth")
	fmt.Println("Auth: " + myAuth)
	myAuthEnvValue := os.Getenv("myAuth")
	if myAuth != myAuthEnvValue {
		http.Error(w, "auth error", 403)
		return
	}

	// upload file
	uploadedFile, handler, err := r.FormFile("data")
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}
	defer uploadedFile.Close()

	// ---------------------------------------------
	// check png
	buffer := make([]byte, 512)
	_, err = uploadedFile.Read(buffer)
	if err != nil {
		fmt.Println(err)
	}
	contentType := http.DetectContentType(buffer)
	fmt.Println("Content Type: " + contentType)

	if contentType != "image/png" {
		http.Error(w, "should be a png image", 403)
		return
	}

	// ---------------------------------------------
	// check file size
	fileSize := handler.Size

	fmt.Printf("The file is %d bytes long", fileSize)
	if fileSize > 8000000 {
		http.Error(w, "shouldn't be larger than 8MB", 403)
		return
	}

	// ---------------------------------------------

	dir, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := handler.Filename

	fileLocation := filepath.Join(dir, "files", filename)
	targetFile, err := os.OpenFile(fileLocation, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, uploadedFile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// --------------------------------
	// connect to db
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	// insert into db
	userSql := "INSERT INTO upload_go_table (type, size, name) VALUES ($1, $2, $3)"

	_, err = db.Exec(userSql, &contentType, &fileSize, &filename)

	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	// ---------------------------------

	w.Write([]byte("done"))
}

// password is intentionally missing for privacy & security
const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = ""
	dbname   = "postgres"
)
