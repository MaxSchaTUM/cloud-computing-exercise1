package main

import (
	"context"
	"fmt"

	// TODO: import template logic from shared/internal package
	// "html/template"
	// "io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// TODO: Remove duplicate Template, loadTemplates, and Render definitions. Use shared package instead.

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

	e := echo.New()
	// e.Renderer = loadTemplates() // TODO: set renderer from shared package

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello from root endpoint! (template logic to be added)")
		// return c.Render(200, "index", nil)
	})

	e.Logger.Fatal(e.Start(":3030"))
}
