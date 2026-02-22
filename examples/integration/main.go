package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kyle-williams-1/bsonic"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	// Get MongoDB connection string from environment or use default
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:password@localhost:27017/bsonic_test?authSource=admin"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("✅ Connected to MongoDB successfully!")

	// Get database and collection
	db := client.Database("bsonic_test")
	collection := db.Collection("users")

	// Create BSON parser
	parser := bsonic.New()

	// Example queries
	queries := []string{
		"name:John Doe",
		"role:admin",
		"active:true",
		"name:J*",
		"profile.location:San Francisco, CA",
		"tags:developer",
		"name:John Doe AND role:admin",
		"name:John Doe OR name:Jane Smith",
		"active:true AND NOT role:admin",
		// Number range examples
		"age:30",
		"age:[25 TO 35]",
		"age:>30",
		"age:<30",
		"age:>=30",
		"age:<=30",
		"age:[25 TO 35] AND active:true",
		"age:>30 OR role:moderator",
	}

	fmt.Println("\n🔍 Testing BSON queries against real MongoDB data:")
	fmt.Println("============================================================")

	for _, queryStr := range queries {
		fmt.Printf("\nQuery: %s\n", queryStr)

		// Parse the query
		bsonQuery, err := parser.Parse(queryStr)
		if err != nil {
			fmt.Printf("❌ Parse error: %v\n", err)
			continue
		}

		// Execute the query
		cursor, err := collection.Find(ctx, bsonQuery)
		if err != nil {
			fmt.Printf("❌ Query error: %v\n", err)
			continue
		}

		// Count results
		var results []bson.M
		if err = cursor.All(ctx, &results); err != nil {
			fmt.Printf("❌ Cursor error: %v\n", err)
			continue
		}

		fmt.Printf("✅ Found %d documents\n", len(results))

		// Show first result if any
		if len(results) > 0 {
			fmt.Printf("   First result: %s (role: %s)\n",
				results[0]["name"],
				results[0]["role"])
		}

		// Show the generated BSON query
		fmt.Printf("   Generated BSON: %+v\n", bsonQuery)
	}

	fmt.Println("\n🎉 Integration example completed successfully!")
	fmt.Println("\nTo run this example:")
	fmt.Println("1. Start MongoDB: make docker-up")
	fmt.Println("2. Run example: go run examples/integration_example.go")
	fmt.Println("3. Stop MongoDB: make docker-down")
}
