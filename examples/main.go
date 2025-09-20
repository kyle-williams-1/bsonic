package main

import (
	"fmt"
	"log"

	"github.com/kyle-williams-1/bsonic"
)

func main() {
	parser := bsonic.New()

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
	}

	fmt.Println("Bsonic - Lucene-style MongoDB BSON Parser")
	fmt.Println("==========================================")
	fmt.Println()

	for _, queryStr := range queries {
		fmt.Printf("Query: %s\n", queryStr)

		query, err := parser.Parse(queryStr)
		if err != nil {
			log.Printf("Error parsing query '%s': %v", queryStr, err)
			continue
		}

		fmt.Printf("BSON: %+v\n", query)
		fmt.Println()
	}
}
