package main

import (
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"os"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
// More on these "tags" like `bson:"_id,omitempty"`: https://go.dev/wiki/Well-known-struct-tags
type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty"`
	ID          string
	BookName    string
	BookAuthor  string
	BookEdition string
	BookPages   string
	BookYear    string
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			ID:          "example1",
			BookName:    "The Vortex",
			BookAuthor:  "José Eustasio Rivera",
			BookEdition: "958-30-0804-4",
			BookPages:   "292",
			BookYear:    "1924",
		},
		{
			ID:          "example2",
			BookName:    "Frankenstein",
			BookAuthor:  "Mary Shelley",
			BookEdition: "978-3-649-64609-9",
			BookPages:   "280",
			BookYear:    "1818",
		},
		{
			ID:          "example3",
			BookName:    "The Black Cat",
			BookAuthor:  "Edgar Allan Poe",
			BookEdition: "978-3-99168-238-7",
			BookPages:   "280",
			BookYear:    "1843",
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":          res.MongoID.Hex(),
			"BookName":    res.BookName,
			"BookAuthor":  res.BookAuthor,
			"BookEdition": res.BookEdition,
			"BookPages":   res.BookPages,
		})
	}

	return ret
}

func getAllBooksForAPI(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"id":      res.ID,
			"title":   res.BookName,
			"author":  res.BookAuthor,
			"pages":   res.BookPages,
			"edition": res.BookEdition,
			"year":    res.BookYear,
		})
	}

	return ret
}

func findAllAuthors(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":         res.MongoID.Hex(),
			"BookAuthor": res.BookAuthor,
		})
	}

	// print ret
	for _, r := range ret {
		fmt.Printf("%+v\n", r)
	}

	return ret
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := os.Getenv("DATABASE_URI")
	if len(uri) == 0 {
		fmt.Printf("failure to load env variable\n")
		os.Exit(1)
	}

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("failed to create client for MongoDB\n")
		os.Exit(1)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf("failed to connect to MongoDB, please make sure the database is running\n")
		os.Exit(1)
	}

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")

	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(LoggerRR)

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		authors := findAllAuthors(coll)
		return c.Render(200, "authors", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "years", books)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	// You will have to expand on the allowed methods for the path
	// `/api/route`, following the common standard.
	// A very good documentation is found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods
	// It specifies the expected returned codes for each type of request
	// method.
	e.GET("/api/books", func(c echo.Context) error {
		books := getAllBooksForAPI(coll)
		return c.JSON(http.StatusOK, books)
	})

	e.POST("/api/books", func(c echo.Context) error {
		// Parse the incoming JSON with the client-side format
		var requestData map[string]string
		if err := c.Bind(&requestData); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		// Extract fields with appropriate validation
		id, hasID := requestData["id"]
		title, hasTitle := requestData["title"]
		author, hasAuthor := requestData["author"]

		// Check for required fields
		if !hasID || id == "" || !hasTitle || title == "" || !hasAuthor || author == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing required fields: id, title, and author are mandatory",
			})
		}

		// Create a BookStore object from the request data
		newBook := BookStore{
			ID:          id,
			BookName:    title,
			BookAuthor:  author,
			BookPages:   requestData["pages"],   // Optional fields
			BookEdition: requestData["edition"], // Optional fields
			BookYear:    requestData["year"],    // Optional fields
		}

		// Check if a book with this ID already exists
		filter := bson.M{"id": newBook.ID}
		existingCount, err := coll.CountDocuments(context.TODO(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to check for existing book",
			})
		}

		// If a book with this ID already exists, return an error
		if existingCount > 0 {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "A book with this ID already exists",
			})
		}

		// Insert the book into the database
		_, err = coll.InsertOne(context.TODO(), newBook)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create book",
			})
		}

		// Format the response in the same way as GET /api/books returns data
		response := map[string]interface{}{
			"id":      newBook.ID,
			"title":   newBook.BookName,
			"author":  newBook.BookAuthor,
			"pages":   newBook.BookPages,
			"edition": newBook.BookEdition,
			"year":    newBook.BookYear,
		}

		// Return 201 Created with the newly created book
		return c.JSON(http.StatusCreated, response)
	})

	e.PUT("/api/books/:id", func(c echo.Context) error {
		// Get the ID from path parameter
		id := c.Param("id")

		// Parse request body
		var requestData map[string]string
		if err := c.Bind(&requestData); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		// Find the book by ID (not MongoID)
		filter := bson.M{"id": id}
		var existingBook BookStore
		err := coll.FindOne(context.TODO(), filter).Decode(&existingBook)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "Book not found",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Database error",
			})
		}

		// Update fields from request
		if title, ok := requestData["title"]; ok && title != "" {
			existingBook.BookName = title
		}
		if author, ok := requestData["author"]; ok && author != "" {
			existingBook.BookAuthor = author
		}
		if edition, ok := requestData["edition"]; ok && edition != "" {
			existingBook.BookEdition = edition
		}
		if pages, ok := requestData["pages"]; ok && pages != "" {
			existingBook.BookPages = pages
		}
		if year, ok := requestData["year"]; ok && year != "" {
			existingBook.BookYear = year
		}

		// Update the book in the database
		_, err = coll.ReplaceOne(context.TODO(), filter, existingBook)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update book",
			})
		}

		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/api/books/:id", func(c echo.Context) error {
		// Get the ID from path parameter
		id := c.Param("id")

		// Create filter to find the book by ID (not MongoID)
		filter := bson.M{"id": id}

		// Perform the deletion
		result, err := coll.DeleteOne(context.TODO(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to delete book",
			})
		}

		// Check if any document was actually deleted
		if result.DeletedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Book not found",
			})
		}

		// Return 200 OK as specified in the requirements
		return c.NoContent(http.StatusOK)
	})

	// We start the server and bind it to port 3030. For future references, this
	// is the application's port and not the external one. For this first exercise,
	// they could be the same if you use a Cloud Provider. If you use ngrok or similar,
	// they might differ.
	// In the submission website for this exercise, you will have to provide the internet-reachable
	// endpoint: http://<host>:<external-port>
	e.Logger.Fatal(e.Start(":3030"))
}

// LoggerRR (Request-Response) is a drop-in replacement for echo/middleware.Logger().
func LoggerRR(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		res := c.Response()

		// ----- Clone request body so handlers can still read it -----
		var reqBody []byte
		if req.Body != nil {
			reqBody, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // restore
		}

		// ----- Wrap the ResponseWriter to capture response body -----
		blw := &bodyLogWriter{ResponseWriter: res.Writer, buf: new(bytes.Buffer)}
		res.Writer = blw

		start := time.Now()
		err := next(c)
		elapsed := time.Since(start)

		// ----- Build nicely formatted output -----
		fmt.Printf(`
────────────────────────────────────────────────────────────
%s %s  (%.0f ms)

Request headers
%s
Request body
%s

→ %d %s
Response headers
%s
Response body
%s
────────────────────────────────────────────────────────────
`,
			req.Method, req.URL.Path, float64(elapsed.Milliseconds()),
			formatHeaders(req.Header),
			string(reqBody),
			res.Status, http.StatusText(res.Status),
			formatHeaders(res.Header()),
			blw.buf.String(),
		)

		return err
	}
}

type bodyLogWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)                   // capture
	return w.ResponseWriter.Write(b) // continue normal write
}

// helper: pretty-print headers
func formatHeaders(h http.Header) string {
	if len(h) == 0 {
		return "(none)\n"
	}
	var sb strings.Builder
	for k, v := range h {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", k, strings.Join(v, ", ")))
	}
	return sb.String()
}
