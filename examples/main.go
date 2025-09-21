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
		// Parentheses examples
		"(name:john OR name:jane) AND age:25",
		"name:john OR (name:jane AND age:25)",
		"((name:john OR name:jane) AND age:25) OR status:active",
		"NOT (name:john OR name:jane)",
		"(name:jo* OR name:ja*) AND (age:25 OR age:30)",
		"created_at:[2023-01-01 TO 2023-12-31] AND (status:active OR status:pending)",
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
