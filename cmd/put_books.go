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
	e.PUT("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		var requestData map[string]string
		if err := c.Bind(&requestData); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

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

		_, err = coll.ReplaceOne(context.TODO(), filter, existingBook)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update book",
			})
		}

		return c.NoContent(http.StatusOK)
	})

	e.Logger.Fatal(e.Start(":3033"))
}
