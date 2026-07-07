package main

import (

	"github.com/joho/godotenv"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/Vikramc25/go-url-shortener/internals/models"
	urlverifier "github.com/davidmytton/url-verifier"
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"

	_ "github.com/lib/pq"
)

func uniqid(prefix string) string {
	now := time.Now()
	sec := now.Unix()
	usec := now.UnixNano() % 0x100000
	return fmt.Sprintf("%s%08x%05x", prefix, sec, usec)
}

func (a *App) GenerateShortenedURL() string {
	var (
		randomChars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321")
		randIntLength = len(randomChars)
		stringLength = 32
	)

	str := make([]rune, stringLength)

	for char := range str {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(randIntLength)))
		if err != nil {
			panic(err)
		}
		str[char] = randomChars[nBig.Int64()]
	}
	hash := sha256.Sum256([]byte(uniqid(string(str))))
	encodedString := base64.StdEncoding.EncodeToString(hash[:])

	return encodedString[0:9]
}

func setErrorInFlash(error string, w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "flash-session")
	if err != nil {
		fmt.Println(err.Error())
	}
	session.AddFlash(error, "error")
	session.Save(r, w)
}

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

type PageData struct {
	BaseURL, Error		string
	URLData				[]*models.ShortenerData
}

type App struct {
	urls *models.ShortenerDataModel
}

func serverError(w http.ResponseWriter, err error){
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func newApp() App {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	schema := `
		CREATE TABLE IF NOT EXISTS urls (
			original_url TEXT PRIMARY KEY,
			shortened_url TEXT UNIQUE NOT NULL,
			clicks INTEGER NOT NULL DEFAULT 0,
			created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_shortened
		ON urls(shortened_url);
		`

	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}

	return App{urls: &models.ShortenerDataModel{DB: db}}
}

var functions = template.FuncMap{
	"formatClicks": formatClicks,
}

func formatClicks(clicks int) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%v", number.Decimal(clicks))
}

func (a *App) getDefaultRoute(w http.ResponseWriter, r *http.Request) {
	tmplFile := "./templates/default.html"
	tmpl, err := template.New("default.html").Funcs(functions).ParseFiles(tmplFile)
	if err != nil {
		fmt.Println(err.Error())
		serverError(w, err)
		return
	}
	urls, err := a.urls.Latest()
	if err != nil {
		serverError(w, err)
		return
	}
	baseURL := "http://" + r.Host + "/"
	pageData := PageData {
		URLData:	urls,
		BaseURL:	baseURL,
	}

	session, err := store.Get(r, "flash-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fm := session.Flashes("error")
	if fm != nil {
		if error, ok := fm[0].(string); ok {
			pageData.Error = error
		} else {
			fmt.Printf("Session flash did not contain an error message. Contained %s.\n", fm[0])
		}
	}

	session.Save(r, w)

	err = tmpl.Execute(w, pageData)
	if err != nil {
		fmt.Println(err.Error())
		serverError(w, err)
	}
}

func (a *App) shortenURL(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err.Error())
		serverError(w, err)
		return
	}

	originalURL := r.PostForm.Get("url")
	if originalURL == "" {
		setErrorInFlash("Please provide a URL to shorten.", w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	verifier := urlverifier.NewVerifier()
	verifier.EnableHTTPCheck()
	result, err := verifier.Verify(originalURL)

	if err != nil {
		fmt.Println(err.Error())
		setErrorInFlash(err.Error(), w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if !result.IsURL {
		fmt.Printf("[%s] is not a valid URL.\n", originalURL)
		setErrorInFlash("Sorry, I can only shorten valid URLs", w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if !result.HTTP.Reachable {
		fmt.Printf("The URL [%s] was not reachable.\n", originalURL)
		setErrorInFlash("The URL was not reachable.", w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	shortenedURL := a.GenerateShortenedURL()
	_, err = a.urls.Insert(originalURL, shortenedURL, 0)
	if err != nil {
		fmt.Println(err.Error())
		setErrorInFlash("We weren't able to shorten the URL.", w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	fmt.Printf("Redirecting to the default route, after shortening %s to %s and persisting it.\n", originalURL, shortenedURL)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) openShortenedRoute(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	shortenedURL := params.ByName("url")

	originalUrl, err := a.urls.Get(shortenedURL)
	if err != nil {
		fmt.Println(err.Error())
		serverError(w, err)
		return
	}

	err = a.urls.IncrementClicks(shortenedURL)
	if err != nil {
		fmt.Println(err.Error())
		serverError(w, err)
		return
	}

	http.Redirect(w, r, originalUrl, http.StatusSeeOther)
}

func (a *App) routes() http.Handler {
	router := httprouter.New()
	fileServer := http.FileServer(http.Dir("./static/"))
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	router.HandlerFunc(http.MethodGet, "/", a.getDefaultRoute)
	router.HandlerFunc(http.MethodPost, "/", a.shortenURL)
	router.HandlerFunc(http.MethodGet, "/o/:url", a.openShortenedRoute)

	standard := alice.New()

	return standard.Then(router)
}


func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8000"
	}
	return port
}

func main() {
	godotenv.Load()
	app := newApp()
	addr := flag.String("addr", ":"+getPort(), "HTTP network address")
	
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	defer app.urls.DB.Close()

	srv := &http.Server{
		Addr:		*addr,
		ErrorLog: 	errorLog,
		Handler: 	app.routes(),
	}

	infoLog.Printf("Starting server on %s", *addr)
	err := srv.ListenAndServe()
	errorLog.Fatal(err)
}
