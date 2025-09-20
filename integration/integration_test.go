//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kyle-williams-1/bsonic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testDB     *mongo.Database
	testClient *mongo.Client
	parser     *bsonic.Parser
)

// TestMain sets up the integration test environment
func TestMain(m *testing.M) {
	// Get MongoDB connection string from environment or use default
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:password@localhost:27017/bsonic_test?authSource=admin"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		fmt.Printf("Failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}

	testClient = client
	testDB = client.Database("bsonic_test")
	parser = bsonic.New()

	// Run tests
	code := m.Run()

	// Cleanup
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Disconnect(ctx)

	os.Exit(code)
}

// TestBasicQueries tests basic field matching queries
func TestBasicQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "exact name match",
			query:    "name:John Doe",
			expected: 1,
		},
		{
			name:     "exact email match",
			query:    "email:jane.smith@example.com",
			expected: 1,
		},
		{
			name:     "role match",
			query:    "role:admin",
			expected: 2,
		},
		{
			name:     "active status match",
			query:    "active:true",
			expected: 4, // Now properly handles boolean values
		},
		{
			name:     "age match",
			query:    "age:30",
			expected: 1, // Now properly handles numeric values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestWildcardQueries tests wildcard pattern matching
func TestWildcardQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "name starts with 'J'",
			query:    "name:J*",
			expected: 3, // John Doe, Jane Smith, Bob Johnson (contains J)
		},
		{
			name:     "email contains 'example'",
			query:    "email:*example*",
			expected: 5, // All users have example.com emails
		},
		{
			name:     "name contains 'o'",
			query:    "name:*o*",
			expected: 4, // John, Bob, Charlie, Alice (contains 'o')
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestDotNotationQueries tests nested field queries
func TestDotNotationQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "profile location match",
			query:    "profile.location:San Francisco, CA",
			expected: 1,
		},
		{
			name:     "profile bio contains 'engineer'",
			query:    "profile.bio:*engineer*",
			expected: 2, // John Doe and Charlie Wilson
		},
		{
			name:     "profile website exists",
			query:    "profile.website:*",
			expected: 4, // All except Bob Johnson
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestArrayQueries tests array field queries
func TestArrayQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "tag contains 'developer'",
			query:    "tags:developer",
			expected: 1,
		},
		{
			name:     "tag contains 'admin' (should not match)",
			query:    "tags:admin",
			expected: 0, // 'admin' is a role, not a tag
		},
		{
			name:     "tag contains 'golang'",
			query:    "tags:golang",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestLogicalOperators tests AND, OR, and NOT operations
func TestLogicalOperators(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "AND operation - active admin",
			query:    "active:true AND role:admin",
			expected: 2, // John Doe and Charlie Wilson (both active and admin)
		},
		{
			name:     "OR operation - John or Jane",
			query:    "name:John Doe OR name:Jane Smith",
			expected: 2,
		},
		{
			name:     "NOT operation - not admin",
			query:    "NOT role:admin",
			expected: 3, // Jane, Bob, Alice
		},
		{
			name:     "Complex - active and not admin",
			query:    "active:true AND NOT role:admin",
			expected: 2, // Jane Smith and Alice Brown (active but not admin)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestProductQueries tests queries on the products collection
func TestProductQueries(t *testing.T) {
	collection := testDB.Collection("products")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "category match",
			query:    "category:electronics",
			expected: 2,
		},
		{
			name:     "in stock products",
			query:    "in_stock:true",
			expected: 2, // Now properly handles boolean values
		},
		{
			name:     "price range (exact match)",
			query:    "price:99.99",
			expected: 1, // Now properly handles numeric values
		},
		{
			name:     "tag contains 'gaming'",
			query:    "tags:gaming",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestComplexQueries tests complex nested queries
func TestComplexQueries(t *testing.T) {
	collection := testDB.Collection("orders")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "customer email match",
			query:    "customer.email:john.doe@example.com",
			expected: 1,
		},
		{
			name:     "order status match",
			query:    "status:completed",
			expected: 1,
		},
		{
			name:     "payment method match",
			query:    "payment_method:credit_card",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestDateQueries tests date-based queries
func TestDateQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "exact date match (using range for same day)",
			query:    "created_at:[2023-01-15 TO 2023-01-16]",
			expected: 1, // John Doe (created 2023-01-15T10:30:00Z)
		},
		{
			name:     "date range - 2023",
			query:    "created_at:[2023-01-01 TO 2023-12-31]",
			expected: 3, // John (2023-01-15), Jane (2023-02-20), Alice (2023-06-05) - Charlie is 2022
		},
		{
			name:     "date greater than 2023-06-01",
			query:    "created_at:>2023-06-01",
			expected: 1, // Alice (created 2023-06-05)
		},
		{
			name:     "date less than 2023-02-01",
			query:    "created_at:<2023-02-01",
			expected: 3, // Charlie (2022-08-30), Bob (2022-11-10), John (2023-01-15)
		},
		{
			name:     "date greater than or equal 2023-02-01",
			query:    "created_at:>=2023-02-01",
			expected: 2, // Jane (2023-02-20), Alice (2023-06-05)
		},
		{
			name:     "date less than or equal 2023-02-01",
			query:    "created_at:<=2023-02-01",
			expected: 3, // Charlie (2022-08-30), Bob (2022-11-10), John (2023-01-15)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestComplexDateQueries tests complex date queries with other conditions
func TestComplexDateQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "date range with role filter",
			query:    "created_at:[2023-01-01 TO 2023-12-31] AND role:admin",
			expected: 1, // Only John (Charlie is 2022)
		},
		{
			name:     "date range with OR condition",
			query:    "created_at:>2023-06-01 OR created_at:<2023-01-01",
			expected: 3, // Alice (after 2023-06-01), Charlie (2022-08-30), Bob (2022-11-10)
		},
		{
			name:     "date range with name filter",
			query:    "created_at:[2023-01-01 TO 2023-12-31] AND name:John*",
			expected: 1, // John Doe
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsonQuery, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
			}

			count, err := collection.CountDocuments(context.Background(), bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if count != int64(tt.expected) {
				t.Errorf("Expected %d documents, got %d for query: %s", tt.expected, count, tt.query)
			}
		})
	}
}

// TestQueryPerformance tests query performance with larger datasets
func TestQueryPerformance(t *testing.T) {
	collection := testDB.Collection("users")

	// Test that queries execute within reasonable time
	start := time.Now()
	bsonQuery, err := parser.Parse("active:true AND role:admin")
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	_, err = collection.CountDocuments(context.Background(), bsonQuery)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	duration := time.Since(start)
	if duration > 1*time.Second {
		t.Errorf("Query took too long: %v", duration)
	}
}

// TestEmptyQuery tests that empty queries return empty BSON
func TestEmptyQuery(t *testing.T) {
	collection := testDB.Collection("users")

	// Empty query should return empty BSON and match all documents
	bsonQuery, err := parser.Parse("")
	if err != nil {
		t.Fatalf("Empty query should not return error: %v", err)
	}

	// Empty BSON should match all documents
	count, err := collection.CountDocuments(context.Background(), bsonQuery)
	if err != nil {
		t.Fatalf("Failed to execute empty query: %v", err)
	}

	// Should match all users (5 total)
	if count != 5 {
		t.Errorf("Expected 5 documents for empty query, got %d", count)
	}
}

// TestQueryValidation tests that invalid queries are handled properly
func TestQueryValidation(t *testing.T) {
	invalidQueries := []string{
		"invalid query format",
		":value",
		"field:",
	}

	for _, query := range invalidQueries {
		t.Run(fmt.Sprintf("invalid_query_%s", query), func(t *testing.T) {
			_, err := parser.Parse(query)
			if err == nil {
				t.Errorf("Expected error for invalid query '%s', got none", query)
			}
		})
	}
}
