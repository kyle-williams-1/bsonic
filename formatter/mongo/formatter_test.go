package mongo_test

import (
	"strings"
	"testing"

	"github.com/kyle-williams-1/bsonic"
)

// TestLuceneMongoFormatterMethods tests MongoFormatter methods
func TestLuceneMongoFormatterMethods(t *testing.T) {
	formatter := bsonic.NewMongoFormatter()

	// Test Format method with invalid AST type
	t.Run("FormatInvalidAST", func(t *testing.T) {
		invalidAST := "not a valid AST"

		_, err := formatter.Format(invalidAST)
		if err == nil {
			t.Fatal("Format should return error for invalid AST type")
		}
		if !strings.Contains(err.Error(), "expected *lucene.ParticipleQuery AST") {
			t.Fatalf("Expected error about AST type, got: %v", err)
		}
	})

	// Test Format method with nil AST
	t.Run("FormatNilAST", func(t *testing.T) {
		_, err := formatter.Format(nil)
		if err == nil {
			t.Fatal("Format should return error for nil AST")
		}
		if !strings.Contains(err.Error(), "expected *lucene.ParticipleQuery AST") {
			t.Fatalf("Expected error about AST type, got: %v", err)
		}
	})
}
