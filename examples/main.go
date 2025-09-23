package main

import (
	"fmt"
	"log"

	"github.com/kyle-williams-1/bsonic"
)

func main() {
	// Example queries
	queries := []string{
		"name:john",
		"name:jo*",
		`name:"john doe"`,
		"name:john AND age:25",
		"name:john OR name:jane",
		"name:john AND NOT age:25",
		"NOT status:inactive",
		"name:jo* OR name:ja* AND NOT age:18",
		"user.profile.email:john@example.com",
		"tags:mongodb",
		// Date query examples
		"created_at:2023-01-15",
		"created_at:2023-01-15T10:30:00Z",
		"created_at:[2023-01-01 TO 2023-12-31]",
		"created_at:>2024-01-01",
		"created_at:<2023-12-31",
		"created_at:>=2024-01-01",
		"created_at:<=2023-12-31",
		"created_at:[2023-01-01 TO 2023-12-31] AND status:active",
		"created_at:>2024-01-01 OR updated_at:<2023-01-01",
		// Numeric range examples
		"age:[18 TO 65]",
		"score:[80 TO 100]",
		"price:[10.50 TO 99.99]",
		"age:[18 TO *]",
		"age:[* TO 65]",
		// Parentheses examples
		"(name:john OR name:jane) AND age:25",
		"name:john OR (name:jane AND age:25)",
		"((name:john OR name:jane) AND age:25) OR status:active",
		"NOT (name:john OR name:jane)",
		"(name:jo* OR name:ja*) AND (age:25 OR age:30)",
		"created_at:[2023-01-01 TO 2023-12-31] AND (status:active OR status:pending)",
		// Text search examples (requires NewWithTextSearch())
		// "engineer software",  // Uncomment to test text search
		// "engineer name:john", // Uncomment to test mixed queries
	}

	fmt.Println("Bsonic - Lucene-style MongoDB BSON Parser")
	fmt.Println("==========================================")
	fmt.Println()

	for _, queryStr := range queries {
		fmt.Printf("Query: %s\n", queryStr)

		query, err := bsonic.Parse(queryStr)
		if err != nil {
			log.Printf("Error parsing query '%s': %v", queryStr, err)
			continue
		}

		fmt.Printf("BSON: %+v\n", query)
		fmt.Println()
	}
}
