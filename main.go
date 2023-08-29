package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

//go:embed templates
var templatesFolder embed.FS

const (
	uploadPath    = "./uploads/"    // Change this to your desired upload folder path
	maxUploadSize = 5 * 1024 * 1024 // 5 MB, adjust as needed
)

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	r := mux.NewRouter()

	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	r.HandleFunc("/upload", uploadHandler)
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/fronc", froncHandler)

	port := ":8080"

	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		os.MkdirAll(uploadPath, os.ModePerm)
	}

	srv := &http.Server{
		Addr: "0.0.0.0" + port,
		// good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	// goroutine so that it doesn't block
	go func() {
		fmt.Printf("Server is running on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	interruptSignal := make(chan os.Signal, 1)

	signal.Notify(interruptSignal, os.Interrupt) // ctrl+c friendly

	<-interruptSignal // block until we receive our signal

	// create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx) // wait until timeout deadline. doesn't block
	log.Println("shutting down")
	os.Exit(0)
}

func froncHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "we fronc!\n")
}

func randfilename(n int, f string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	split := strings.Split(f, ".")
	extension := split[len(split)-1] // get the last element as the extension
	return string(b) + "." + extension
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form data with a specified max memory limit (in bytes)
	r.ParseMultipartForm(10 << 20) // 10 MB max in-memory size

	// Get the uploaded file
	file, handler, err := r.FormFile("file") // "file" should match the name attribute in your HTML form
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create or open a new file in the desired directory
	// Replace "/path/to/your/directory" with your actual directory path
	genfilename := randfilename(6, handler.Filename)
	newFile, err := os.Create("./uploads/" + genfilename)
	if err != nil {
		fmt.Println("Error creating the file:", err)
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Copy the uploaded file data to the new file
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("Error copying file data:", err)
		http.Error(w, "Error copying file data", http.StatusInternalServerError)
		return
	}

	// fmt.Fprintln(w, "File uploaded successfully:", handler.Filename)
	// http.Redirect(w, r, "http://localhost:8080/" + genfilename, http.StatusSeeOther)
	http.Redirect(w, r, "/fronc", http.StatusSeeOther)
}
