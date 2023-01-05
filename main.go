package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"my-project/config"

	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gosimple/slug"
)

func main() {
	route := mux.NewRouter()

	// Connect to Database
	config.DatabaseConnect()

	// for public folder
	// ex: localhost:port/public/ +../path/to/file
	route.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))

	route.HandleFunc("/home", home).Methods("GET")
	route.HandleFunc("/contact", contact).Methods("GET")

	// CRUD Project
	// create
	route.HandleFunc("/create", createProject).Methods("GET")
	route.HandleFunc("/create", storeProject).Methods("POST")
	// read
	route.HandleFunc("/detail/{id}", detailProject).Methods("GET")
	// update
	route.HandleFunc("/edit/{id}", editProject).Methods("GET")
	route.HandleFunc("/edit/{id}", updateProject).Methods("POST")
	// delete
	route.HandleFunc("/delete/{id}", deleteProject).Methods("GET")

	port := "9000"
	fmt.Println("Server berjalan pada port", port)
	http.ListenAndServe("localhost:"+port, route)
}

// Project Struct
type Project struct {
	ID           int
	ProjectName  string
	StartDate    time.Time
	EndDate      time.Time
	Duration     string
	Description  string
	Technologies []string
	Image        string
}

// home
func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	tmpt, err := template.ParseFiles("views/index.html")

	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	// Query to database
	rowsData, err := config.Conn.Query(context.Background(), "SELECT * FROM tb_projects ORDER BY id DESC")
	if err != nil {
		fmt.Println("Message : " + err.Error())
		return
	}

	// Project slice to hold data from returned rowsData.
	var resultData []Project

	// Loop through rowsData, using Scan to assign column data to struct fields.
	for rowsData.Next() {
		var each = Project{}
		err := rowsData.Scan(&each.ID, &each.ProjectName, &each.StartDate, &each.EndDate, &each.Technologies, &each.Description, &each.Image)
		if err != nil {
			fmt.Println("Message : " + err.Error())
			return
		}
		// add Duration result from calc of StartDate and EndDate
		each.Duration = config.GetDurationTime(each.StartDate, each.EndDate)
		// Append to Project slice
		resultData = append(resultData, each)
	}
	// for more database query: https://go.dev/doc/database/querying

	// fmt.Println(resultData)
	data := map[string]interface{}{
		"Projects": resultData,
	}

	tmpt.Execute(w, data)
}

//
// CRUD Project
//

// create - createProject
func createProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	tmpt, err := template.ParseFiles("views/create-project.html")

	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	tmpt.Execute(w, nil)
}

// create - storeProject
func storeProject(w http.ResponseWriter, r *http.Request) {
	// left shift 32 << 20 which results in 32*2^20 = 33554432
	// x << y, results in x*2^y
	// this is for maxMemory to temporary save data form.
	err := r.ParseMultipartForm(32 << 20)

	if err != nil {
		log.Fatal(err)
	}

	project_name := r.PostForm.Get("project_name")
	technologies := r.Form["technologies"]
	description := r.PostForm.Get("description")

	// Image
	// Retrieve the image from form data
	uploadedFile, handler, err := r.FormFile("image")
	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	defer uploadedFile.Close()
	fileLocation := "public/uploads/"
	imageName := slug.Make(project_name)
	_ = os.MkdirAll(fileLocation, os.ModePerm)
	fullPath := fileLocation + imageName + filepath.Ext(handler.Filename)
	targetFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	defer targetFile.Close()
	// Copy the file to the destination path
	_, err = io.Copy(targetFile, uploadedFile)
	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	// Image Path
	image_path := fileLocation + imageName + filepath.Ext(handler.Filename)
	// End For Image

	// Date
	const (
		layoutISO = "2006-01-02"
	)
	start_date, _ := time.Parse(layoutISO, r.PostForm.Get("start_date"))
	end_date, _ := time.Parse(layoutISO, r.PostForm.Get("end_date"))

	_, err = config.Conn.Exec(context.Background(), "INSERT INTO tb_projects(project_name, start_date, end_date, technologies, description, image) VALUES ($1, $2, $3, $4, $5, $6)", project_name, start_date, end_date, technologies, description, image_path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

// read
// detailProject
func detailProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	tmpt, err := template.ParseFiles("views/detail-project.html")

	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var resultData = Project{}

	err = config.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", id).Scan(
		&resultData.ID, &resultData.ProjectName, &resultData.StartDate, &resultData.EndDate, &resultData.Technologies, &resultData.Description, &resultData.Image,
	)
	// add Duration result from calc of StartDate and EndDate
	resultData.Duration = config.GetDurationTime(resultData.StartDate, resultData.EndDate)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	// fmt.Println(resultData)

	data := map[string]interface{}{
		"Project": resultData,
	}
	tmpt.Execute(w, data)
}

// update - editProject
func editProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	tmpt, err := template.ParseFiles("views/edit-project.html")

	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var resultData = Project{}

	err = config.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", id).Scan(
		&resultData.ID, &resultData.ProjectName, &resultData.StartDate, &resultData.EndDate, &resultData.Technologies, &resultData.Description, &resultData.Image,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	// fmt.Println(resultData)

	data := map[string]interface{}{
		"Project": resultData,
	}
	tmpt.Execute(w, data)
}

// update - updateProject
func updateProject(w http.ResponseWriter, r *http.Request) {

	// left shift 32 << 20 which results in 32*2^20 = 33554432
	// x << y, results in x*2^y
	err := r.ParseMultipartForm(32 << 20)

	if err != nil {
		log.Fatal(err)
	}

	project_name := r.PostForm.Get("project_name")
	technologies := r.Form["technologies"]
	description := r.PostForm.Get("description")

	// Image
	// Retrieve the image from form data
	uploadedFile, handler, err := r.FormFile("image")
	if err != nil {
		w.Write([]byte("Error message upload file: " + err.Error()))
		return
	}
	defer uploadedFile.Close()
	fileLocation := "public/uploads/"
	imageName := slug.Make(project_name)
	_ = os.MkdirAll(fileLocation, os.ModePerm)
	fullPath := fileLocation + imageName + filepath.Ext(handler.Filename)
	targetFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		w.Write([]byte("Error message target file: " + err.Error()))
		return
	}
	defer targetFile.Close()
	// Copy the file to the destination path
	_, err = io.Copy(targetFile, uploadedFile)
	if err != nil {
		w.Write([]byte("Error message copy file: " + err.Error()))
		return
	}
	// Image Path
	image_path := fileLocation + imageName + filepath.Ext(handler.Filename)
	// End For Image
	// Date
	const (
		layoutISO = "2006-01-02"
	)
	start_date, _ := time.Parse(layoutISO, r.PostForm.Get("start_date"))
	end_date, _ := time.Parse(layoutISO, r.PostForm.Get("end_date"))
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	_, err = config.Conn.Exec(context.Background(), "UPDATE tb_projects SET project_name = $1, start_date = $2, end_date = $3, technologies = $4, description = $5, image = $6 WHERE id = $7", project_name, start_date, end_date, technologies, description, image_path, id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

// delete
func deleteProject(w http.ResponseWriter, r *http.Request) {

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	_, err := config.Conn.Exec(context.Background(), "DELETE FROM tb_projects WHERE id=$1", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Message: " + err.Error()))
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// contact
func contact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	tmpt, err := template.ParseFiles("views/contact.html")

	if err != nil {
		w.Write([]byte("Message: " + err.Error()))
		return
	}

	tmpt.Execute(w, nil)
}
