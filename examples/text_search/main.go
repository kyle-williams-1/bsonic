package main

import (
	"fmt"
	"log"

	"github.com/kyle-williams-1/bsonic"
)

func main() {
	fmt.Println("Bsonic - Text Search Examples")
	fmt.Println("=============================")
	fmt.Println()

	// Create a parser with text search enabled
	parser := bsonic.NewWithTextSearch()

	// Example queries
	queries := []string{
		// Pure text search queries
		"engineer software",
		"designer user interface",
		"devops infrastructure",
		"data scientist machine learning",

		// Mixed queries (text search + field search)
		"engineer name:john",
		"software role:admin",
		"designer active:true",
		"devops age:>25",

		// Complex mixed queries
		"software engineer role:admin AND active:true",
		"designer (role:user OR role:moderator)",
		"devops name:charlie OR name:david",
		"data scientist (role:admin AND age:>30) OR (role:senior AND experience:>5)",
	}

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

	// Example with text search disabled (should error)
	fmt.Println("Text search disabled example:")
	fmt.Println("=============================")

	disabledParser := bsonic.New()
	_, err := disabledParser.Parse("engineer software")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}
}
