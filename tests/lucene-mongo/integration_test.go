//go:build integration

package lucene_mongo_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kyle-williams-1/bsonic"
	bsonic_config "github.com/kyle-williams-1/bsonic/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
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

	// Create parser with default fields for integration tests
	cfg := bsonic_config.Default().WithDefaultFields([]string{"name", "description", "email"})
	parser, err = bsonic.NewWithConfig(cfg)
	if err != nil {
		fmt.Printf("Failed to create parser: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Disconnect(ctx)

	os.Exit(code)
}

// TestBasicQueries tests basic field matching, wildcard patterns, and nested field queries
func TestBasicQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		// Basic field matching
		{
			name:     "exact name match",
			query:    "name:\"John Doe\"",
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
			expected: 4, // BSON library correctly parses boolean values
		},
		{
			name:     "age match",
			query:    "age:30",
			expected: 1, // BSON library correctly parses numeric values
		},
		// Wildcard pattern matching
		{
			name:     "name starts with 'J'",
			query:    "name:J*",
			expected: 2, // John Doe, Jane Smith (starts with J)
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
		{
			name:     "name ends with 'son'",
			query:    "name:*son",
			expected: 2, // Johnson, Wilson (ends with 'son')
		},
		// Nested field queries (dot notation)
		{
			name:     "profile location match",
			query:    "profile.location:\"San Francisco, CA\"",
			expected: 1,
		},
		{
			name:     "profile bio contains 'engineer' (case-sensitive)",
			query:    "profile.bio:*engineer*",
			expected: 1, // John Doe only (lowercase 'engineer'); Charlie has 'Engineer' with capital E
		},
		{
			name:     "profile website exists",
			query:    "profile.website:*",
			expected: 4, // All except Bob Johnson
		},
		// ID field conversion tests
		{
			name:     "id field conversion to _id with ObjectID",
			query:    "id:507f1f77bcf86cd799439011",
			expected: 0, // No user with this specific ObjectID
		},
		{
			name:     "nested id field conversion to _id with ObjectID",
			query:    "user.id:507f1f77bcf86cd799439011",
			expected: 0, // No user with this specific ObjectID
		},
		{
			name:     "user_id field with ObjectID",
			query:    "user_id:507f1f77bcf86cd799439011",
			expected: 0, // No user with this specific ObjectID
		},
		{
			name:     "order_id field with non-ObjectID string",
			query:    "order_id:12345",
			expected: 0, // Should fallback to string search
		},
		{
			name:     "product_id field with short string",
			query:    "product_id:abc",
			expected: 0, // Should fallback to string search (not 24 chars)
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

// TestLogicalQueries tests logical operators and complex query combinations
func TestLogicalQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		// Basic logical operators
		{
			name:     "AND operation - active admin",
			query:    "active:true AND role:admin",
			expected: 2, // John Doe and Charlie Wilson (both active and admin)
		},
		{
			name:     "OR operation - John or Jane",
			query:    "name:\"John Doe\" OR name:\"Jane Smith\"",
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
		// Parentheses grouping and precedence control
		{
			name:     "simple grouping - OR with AND",
			query:    "(name:\"John Doe\" OR name:\"Jane Smith\") AND active:true",
			expected: 2, // Both John and Jane are active
		},
		{
			name:     "NOT with grouped expression",
			query:    "NOT (role:admin OR role:moderator)",
			expected: 2, // Jane Smith and Bob Johnson (neither admin nor moderator)
		},
		{
			name:     "nested parentheses",
			query:    "((name:\"John Doe\" OR name:\"Jane Smith\") AND active:true) OR role:moderator",
			expected: 3, // John, Jane (active) + Bob (moderator)
		},
		{
			name:     "complex precedence override",
			query:    "name:\"John Doe\" OR (name:\"Jane Smith\" AND active:true)",
			expected: 2, // John (any condition) + Jane (if active)
		},
		{
			name:     "grouped NOT operations",
			query:    "(NOT role:admin) AND (NOT name:\"Bob Johnson\")",
			expected: 2, // Alice Brown and Jane Smith (not admin and not Bob)
		},
		{
			name:     "multiple OR groups with AND",
			query:    "(name:\"John Doe\" OR name:\"Jane Smith\") AND (active:true OR role:moderator)",
			expected: 2, // John (active) + Jane (active)
		},
		// Additional logical operator edge cases
		{
			name:     "triple AND condition",
			query:    "active:true AND role:admin AND name:\"John Doe\"",
			expected: 1, // Only John Doe matches all three
		},
		{
			name:     "triple OR condition",
			query:    "name:\"John Doe\" OR name:\"Jane Smith\" OR name:\"Bob Johnson\"",
			expected: 3,
		},
		{
			name:     "mixed precedence without parentheses",
			query:    "active:true AND role:admin OR name:\"Alice Brown\"",
			expected: 3, // John, Charlie (active admin) + Alice (any condition)
		},
		{
			name:     "NOT with multiple conditions",
			query:    "NOT (active:false OR role:moderator)",
			expected: 3, // John, Jane, Alice (not inactive and not moderator)
		},
		{
			name:     "complex nested grouping",
			query:    "((name:\"John Doe\" OR name:\"Charlie Wilson\") AND role:admin) OR (name:\"Alice Brown\" AND role:moderator)",
			expected: 3, // John, Charlie (admin) + Alice (moderator)
		},
		// NOT with wildcards and regex patterns
		{
			name:     "NOT with wildcard - exclude names starting with 'J'",
			query:    "NOT name:J*",
			expected: 3, // Bob Johnson, Alice Brown, Charlie Wilson (excludes John Doe, Jane Smith)
		},
		{
			name:     "NOT with regex - exclude names matching pattern",
			query:    "NOT name:/^J.*/",
			expected: 3, // Bob Johnson, Alice Brown, Charlie Wilson (excludes John Doe, Jane Smith)
		},
		{
			name:     "NOT with comparison operator - exclude age > 30",
			query:    "NOT age:>30",
			expected: 3, // John (30), Jane (28), Alice (25) - excludes Bob (35), Charlie (42)
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

// TestCollectionSpecificQueries tests queries on different collections
func TestCollectionSpecificQueries(t *testing.T) {
	// Test products collection queries
	t.Run("CollectionSpecificQueries", func(t *testing.T) {
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
	})
}

// TestDateQueries tests date-based queries including complex combinations
func TestDateQueries(t *testing.T) {
	collection := testDB.Collection("users")

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		// Basic date queries
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
		// Complex date queries with other conditions
		{
			name:     "exact date match with parentheses",
			query:    "(created_at:[2023-01-15 TO 2023-01-16]) AND active:true",
			expected: 1, // John Doe (created 2023-01-15T10:30:00Z)
		},
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

// TestNumberRangeQueries tests number range queries including complex combinations
func TestNumberRangeQueries(t *testing.T) {
	// Test user age queries
	t.Run("UserAgeQueries", func(t *testing.T) {
		collection := testDB.Collection("users")

		tests := []struct {
			name     string
			query    string
			expected int
		}{
			// Basic age range queries
			{
				name:     "age range 25-35",
				query:    "age:[25 TO 35]",
				expected: 4, // John (30), Jane (28), Bob (35), Alice (25)
			},
			{
				name:     "age greater than 30",
				query:    "age:>30",
				expected: 2, // Bob (35), Charlie (42)
			},
			{
				name:     "age less than 30",
				query:    "age:<30",
				expected: 2, // Jane (28), Alice (25)
			},
			{
				name:     "age greater than or equal 30",
				query:    "age:>=30",
				expected: 3, // John (30), Bob (35), Charlie (42)
			},
			{
				name:     "age less than or equal 30",
				query:    "age:<=30",
				expected: 3, // John (30), Jane (28), Alice (25)
			},
			{
				name:     "age range with wildcard start",
				query:    "age:[* TO 30]",
				expected: 3, // John (30), Jane (28), Alice (25)
			},
			{
				name:     "age range with wildcard end",
				query:    "age:[30 TO *]",
				expected: 3, // John (30), Bob (35), Charlie (42)
			},
			// Complex age queries with other conditions
			{
				name:     "age range with role filter",
				query:    "age:[25 TO 35] AND role:admin",
				expected: 1, // John (age 30, role admin)
			},
			{
				name:     "age greater than 30 OR role moderator",
				query:    "age:>30 OR role:moderator",
				expected: 3, // Bob (35), Charlie (42), Alice (moderator)
			},
			{
				name:     "age range with active status",
				query:    "age:[25 TO 35] AND active:true",
				expected: 3, // John (30, active), Jane (28, active), Alice (25, active) - all in range 25-35
			},
			{
				name:     "age less than 30 AND active status",
				query:    "age:<30 AND active:true",
				expected: 2, // Jane (28, active), Alice (25, active)
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
	})

	// Test product price queries
	t.Run("ProductPriceQueries", func(t *testing.T) {
		collection := testDB.Collection("products")

		tests := []struct {
			name     string
			query    string
			expected int
		}{
			{
				name:     "price range 50-100",
				query:    "price:[50 TO 100]",
				expected: 2, // Wireless Headphones (99.99), Gaming Mouse (79.99)
			},
			{
				name:     "price greater than 80",
				query:    "price:>80",
				expected: 1, // Wireless Headphones (99.99)
			},
			{
				name:     "price less than 20",
				query:    "price:<20",
				expected: 1, // Coffee Mug (15.99)
			},
			{
				name:     "price greater than or equal 80",
				query:    "price:>=80",
				expected: 1, // Wireless Headphones (99.99) - Gaming Mouse is 79.99 < 80
			},
			{
				name:     "price less than or equal 80",
				query:    "price:<=80",
				expected: 2, // Gaming Mouse (79.99), Coffee Mug (15.99)
			},
			{
				name:     "price range with wildcard start",
				query:    "price:[* TO 80]",
				expected: 2, // Gaming Mouse (79.99), Coffee Mug (15.99)
			},
			{
				name:     "price range with wildcard end",
				query:    "price:[80 TO *]",
				expected: 1, // Wireless Headphones (99.99)
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
	})
}

// TestRegexPatterns tests regex pattern matching functionality
func TestRegexPatterns(t *testing.T) {
	testCases := []struct {
		name       string
		query      string
		expected   int
		collection string
	}{
		// Basic regex patterns on user names
		{
			name:       "basic regex pattern - john doe",
			query:      "name:/John Doe/",
			expected:   1, // John Doe
			collection: "users",
		},
		{
			name:       "anchored regex pattern - starts with john",
			query:      "name:/^John.*/",
			expected:   1, // John Doe
			collection: "users",
		},
		{
			name:       "regex pattern - jane",
			query:      "name:/.*Jane.*/",
			expected:   1, // Jane Smith
			collection: "users",
		},
		{
			name:       "regex pattern with alternation - admin or moderator",
			query:      "role:/^(admin|moderator)$/",
			expected:   3, // John Doe (admin), Alice Brown (moderator), Charlie Wilson (admin)
			collection: "users",
		},
		// Email regex patterns
		{
			name:       "email regex pattern - example.com domain",
			query:      "email:/.*@example\\.com$/",
			expected:   5, // All users have example.com emails
			collection: "users",
		},
		{
			name:       "email regex pattern - john or jane emails",
			query:      "email:/^(john|jane)\\..*/",
			expected:   2, // John Doe and Jane Smith
			collection: "users",
		},
		// Tag regex patterns
		{
			name:       "tag regex pattern - contains 'dev'",
			query:      "tags:/.*dev.*/",
			expected:   2, // John Doe (developer), Charlie Wilson (devops)
			collection: "users",
		},
		{
			name:       "tag regex pattern - starts with 'g'",
			query:      "tags:/^g.*/",
			expected:   1, // John Doe (golang)
			collection: "users",
		},
		// Profile bio regex patterns
		{
			name:       "profile bio regex pattern - contains 'engineer'",
			query:      "profile.bio:/.*engineer.*/",
			expected:   1, // John Doe (Senior software engineer) - case sensitive
			collection: "users",
		},
		{
			name:       "profile bio regex pattern - contains 'designer'",
			query:      "profile.bio:/.*Designer.*/",
			expected:   1, // Jane Smith (UX/UI Designer)
			collection: "users",
		},
		// Website regex patterns
		{
			name:       "website regex pattern - https websites",
			query:      "profile.website:/^https:.*/",
			expected:   4, // John, Jane, Alice, Charlie (Bob has null website)
			collection: "users",
		},
		{
			name:       "website regex pattern - .dev domains",
			query:      "profile.website:/.*\\.dev$/",
			expected:   1, // John Doe (https://johndoe.dev)
			collection: "users",
		},
		// Product regex patterns
		{
			name:       "product name regex pattern - wireless",
			query:      "name:/.*Wireless.*/",
			expected:   1, // Wireless Headphones
			collection: "products",
		},
		{
			name:       "product category regex pattern - electronics",
			query:      "category:/^electronics$/",
			expected:   2, // Wireless Headphones, Gaming Mouse
			collection: "products",
		},
		{
			name:       "product tag regex pattern - audio related",
			query:      "tags:/^audio$/",
			expected:   1, // Wireless Headphones
			collection: "products",
		},
		// Complex regex patterns with logical operators
		{
			name:       "regex with AND condition",
			query:      "name:/John Doe/ AND active:true",
			expected:   1, // John Doe
			collection: "users",
		},
		{
			name:       "regex with OR condition",
			query:      "name:/John Doe/ OR name:/Jane Smith/",
			expected:   2, // John Doe and Jane Smith
			collection: "users",
		},
		{
			name:       "regex with NOT condition",
			query:      "name:/John Doe/ AND NOT role:guest",
			expected:   1, // John Doe (not a guest)
			collection: "users",
		},
		{
			name:       "regex with grouping and OR condition",
			query:      "(name:/.*John.*/ OR name:/.*Jane.*/) AND active:true",
			expected:   2, // John Doe and Jane Smith (both active)
			collection: "users",
		},
		// Edge cases and special characters
		{
			name:       "regex with escaped characters",
			query:      "profile.website:/.*\\.(dev|design|blog|tech)$/",
			expected:   4, // All users with websites
			collection: "users",
		},
		{
			name:       "regex with digit matching",
			query:      "profile.bio:/.*\\d+.*/",
			expected:   0, // No bios contain numbers
			collection: "users",
		},
		{
			name:       "regex with word boundaries",
			query:      "tags:/.*dev.*/",
			expected:   2, // John Doe (developer), Charlie Wilson (devops)
			collection: "users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the query
			bsonQuery, err := parser.Parse(tc.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tc.query, err)
			}

			// Execute the query
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cursor, err := testDB.Collection(tc.collection).Find(ctx, bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}
			defer cursor.Close(ctx)

			var results []bson.M
			if err = cursor.All(ctx, &results); err != nil {
				t.Fatalf("Failed to decode results: %v", err)
			}

			count := len(results)
			if count != tc.expected {
				t.Errorf("Expected %d documents for query '%s', got %d", tc.expected, tc.query, count)
				t.Logf("Query BSON: %+v", bsonQuery)
				t.Logf("Results: %+v", results)
			}
		})
	}
}

// TestDefaultFieldsSearch tests default fields search functionality
func TestDefaultFieldsSearch(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "simple default fields search",
			query:    `"John Doe"`,
			expected: 1, // Should match John Doe in name field
		},
		{
			name:     "multiple fields search unquoted",
			query:    `John* Doe`,
			expected: 1, // Should match John Doe but not Bob Johnson
		},
		{
			name:     "default fields search with single quotes",
			query:    `'Jane Smith'`,
			expected: 1, // Should match Jane Smith in name field
		},
		{
			name:     "default fields search with field query",
			query:    `"John Doe" AND active:true`,
			expected: 1, // Should match John Doe who is active
		},
		{
			name:     "default fields search with OR condition",
			query:    `"John Doe" AND (active:true OR role:admin)`,
			expected: 1, // Should match John Doe who is active
		},
		{
			name:     "multiple default fields searches with OR",
			query:    `("John Doe" OR "Jane Smith") AND active:true`,
			expected: 2, // Should match both John Doe and Jane Smith who are active
		},
		{
			name:     "default fields search with NOT condition",
			query:    `"John Doe" AND NOT role:guest`,
			expected: 1, // Should match John Doe who is not a guest
		},
		{
			name:     "unquoted single word default fields search (case sensitive)",
			query:    `*John*`,
			expected: 2, // Should match John Doe and Bob Johnson
		},
		{
			name:     "unquoted multiple words with partial match wildcards",
			query:    `*John* *Doe*`,
			expected: 2, // Should match "John Doe" and "Bob Johnson" (both contain John)
		},
		{
			name:     "unquoted default fields search with field query (case sensitive)",
			query:    `*John* AND active:true`,
			expected: 1, // Should match John Doe who is active
		},
		{
			name:     "unquoted default fields search with OR condition",
			query:    `John* AND (active:true OR role:admin)`,
			expected: 1, // Should match John Doe who is active
		},
		{
			name:     "multiple unquoted default fields searches with OR",
			query:    `(*John* OR *Jane*) AND active:true`,
			expected: 2, // Should match both John Doe and Jane Smith who are active
		},
		{
			name:     "mixed quoted and unquoted default fields searches",
			query:    `("John Doe" OR *Jane*) AND active:true`,
			expected: 2, // Should match both John Doe and Jane Smith who are active
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the query
			bsonQuery, err := parser.Parse(tc.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", tc.query, err)
			}

			// Execute the query
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cursor, err := testDB.Collection("users").Find(ctx, bsonQuery)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}
			defer cursor.Close(ctx)

			var results []bson.M
			if err = cursor.All(ctx, &results); err != nil {
				t.Fatalf("Failed to decode results: %v", err)
			}

			count := len(results)
			if count != tc.expected {
				t.Errorf("Expected %d documents for query '%s', got %d", tc.expected, tc.query, count)
				t.Logf("Query BSON: %+v", bsonQuery)
				t.Logf("Results: %+v", results)
			}
		})
	}
}

// TestUtilityAndEdgeCases tests utility functions and edge cases
func TestUtilityAndEdgeCases(t *testing.T) {
	// Test empty query handling
	t.Run("EmptyQuery", func(t *testing.T) {
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
	})

	// Test query validation
	t.Run("QueryValidation", func(t *testing.T) {
		invalidQueries := []string{
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
	})

}

// TestDefaultFieldsIntegration tests default fields functionality with actual MongoDB queries
func TestDefaultFieldsIntegration(t *testing.T) {
	collection := testDB.Collection("users")
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		defaultFields []string
		expected      int
	}{
		{
			name:          "single default field name search",
			query:         "*John*",
			defaultFields: []string{"name"},
			expected:      2, // Should match "John Doe" and "Bob Johnson"
		},
		{
			name:          "single default field email search",
			query:         "*jane*",
			defaultFields: []string{"email"},
			expected:      1, // Should match jane.smith@example.com (email is lowercase)
		},
		{
			name:          "multiple default fields search",
			query:         "admin",
			defaultFields: []string{"name", "role"},
			expected:      2, // Should match "John Doe" and "Charlie Wilson" (both have role:admin)
		},
		{
			name:          "default field with wildcard",
			query:         "J*",
			defaultFields: []string{"name"},
			expected:      2, // Should match "John Doe" and "Jane Smith"
		},
		{
			name:          "default field with field query",
			query:         "*John* AND active:true",
			defaultFields: []string{"name"},
			expected:      1, // Should match "John Doe" who is active
		},
		{
			name:          "default field with OR condition",
			query:         "*John* AND (role:admin OR role:user)",
			defaultFields: []string{"name"},
			expected:      2, // Should match "John Doe" (admin) and "Bob Johnson" (user)
		},
		// ID field conversion with custom config
		{
			name:          "id field without conversion (disabled)",
			query:         "id:507f1f77bcf86cd799439011",
			defaultFields: []string{"name"},
			expected:      0, // No user with this specific string ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter bson.M
			var err error

			// Special handling for ID field conversion with custom config
			if tt.name == "id field without conversion (disabled)" {
				cfg := bsonic_config.Default().
					WithDefaultFields(tt.defaultFields).
					WithReplaceIDWithMongoID(false).
					WithAutoConvertIDToObjectID(false)

				parser, err := bsonic.NewWithConfig(cfg)
				if err != nil {
					t.Fatalf("NewWithConfig should not return error, got: %v", err)
				}

				filter, err = parser.Parse(tt.query)
				if err != nil {
					t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
				}

				// Check that field name was NOT converted
				if _, exists := filter["id"]; !exists {
					t.Fatalf("Expected 'id' field, got: %+v", filter)
				}

				// Value should remain as string when conversion is disabled
				strValue, ok := filter["id"].(string)
				if !ok {
					t.Fatalf("Expected string, got %T: %+v", filter["id"], filter["id"])
				}
				if strValue != "507f1f77bcf86cd799439011" {
					t.Fatalf("Expected string value '507f1f77bcf86cd799439011', got '%s'", strValue)
				}
			} else {
				// Parse the query with default fields
				filter, err = bsonic.ParseWithDefaults(tt.defaultFields, tt.query)
				if err != nil {
					t.Fatalf("Failed to parse query '%s': %v", tt.query, err)
				}
			}

			// Execute the query
			count, err := collection.CountDocuments(ctx, filter)
			if err != nil {
				t.Fatalf("Failed to execute query '%s': %v", tt.query, err)
			}

			if int(count) != tt.expected {
				t.Errorf("Query '%s' with default fields %v: expected %d documents, got %d", tt.query, tt.defaultFields, tt.expected, count)
			}
		})
	}
}
