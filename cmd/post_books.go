package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type BookStore struct {
	ID          string `bson:"id"`
	BookName    string `bson:"bookname"`
	BookAuthor  string `bson:"bookauthor"`
	BookEdition string `bson:"bookedition"`
	BookPages   string `bson:"bookpages"`
	BookYear    string `bson:"bookyear"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := os.Getenv("DATABASE_URI")
	if len(uri) == 0 {
		fmt.Printf("failure to load env variable\n")
		os.Exit(1)
	}

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

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	coll := client.Database("exercise-1").Collection("information")

	e := echo.New()
	e.POST("/api/books", func(c echo.Context) error {
		var requestData map[string]string
		if err := c.Bind(&requestData); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		id, hasID := requestData["id"]
		title, hasTitle := requestData["title"]
		author, hasAuthor := requestData["author"]

		if !hasID || id == "" || !hasTitle || title == "" || !hasAuthor || author == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing required fields: id, title, and author are mandatory",
			})
		}

		newBook := BookStore{
			ID:          id,
			BookName:    title,
			BookAuthor:  author,
			BookPages:   requestData["pages"],
			BookEdition: requestData["edition"],
			BookYear:    requestData["year"],
		}

		filter := bson.M{"id": newBook.ID}
		existingCount, err := coll.CountDocuments(context.TODO(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to check for existing book",
			})
		}

		if existingCount > 0 {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "A book with this ID already exists",
			})
		}

		_, err = coll.InsertOne(context.TODO(), newBook)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create book",
			})
		}

		response := map[string]interface{}{
			"id":      newBook.ID,
			"title":   newBook.BookName,
			"author":  newBook.BookAuthor,
			"pages":   newBook.BookPages,
			"edition": newBook.BookEdition,
			"year":    newBook.BookYear,
		}

		return c.JSON(http.StatusCreated, response)
	})

	e.Logger.Fatal(e.Start(":3032"))
}
