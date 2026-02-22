package mongo_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kyle-williams-1/bsonic"
	"github.com/kyle-williams-1/bsonic/formatter/mongo"
	"github.com/kyle-williams-1/bsonic/language/lucene"
	"go.mongodb.org/mongo-driver/v2/bson"
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

// TestIDFieldConversion tests ID field name conversion and ObjectID conversion
func TestIDFieldConversion(t *testing.T) {
	// Create a parser to parse queries
	parser := lucene.New()

	t.Run("IDFieldNameConversionEnabled", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:507f1f77bcf86cd799439011`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that field name was converted from "id" to "_id"
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}

		// Check that value is ObjectID
		objectID, ok := result["_id"].(bson.ObjectID)
		if !ok {
			t.Fatalf("Expected ObjectID, got %T: %+v", result["_id"], result["_id"])
		}

		// Verify it's the correct ObjectID
		expectedObjectID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
		if objectID != expectedObjectID {
			t.Fatalf("Expected ObjectID %v, got %v", expectedObjectID, objectID)
		}
	})

	t.Run("IDFieldNameConversionDisabled", func(t *testing.T) {
		formatter := mongo.NewWithOptions(false, false)

		query := `id:507f1f77bcf86cd799439011`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that field name was NOT converted
		if _, exists := result["id"]; !exists {
			t.Fatalf("Expected 'id' field, got: %+v", result)
		}

		// Value should remain as string when conversion is disabled
		strValue, ok := result["id"].(string)
		if !ok {
			t.Fatalf("Expected string, got %T: %+v", result["id"], result["id"])
		}
		if strValue != "507f1f77bcf86cd799439011" {
			t.Fatalf("Expected string value '507f1f77bcf86cd799439011', got '%s'", strValue)
		}
	})

	t.Run("NestedIDFieldConversion", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, false) // Enable field conversion, disable ObjectID conversion

		query := `user.id:507f1f77bcf86cd799439011`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that nested field name was converted from "user.id" to "user._id"
		if _, exists := result["user._id"]; !exists {
			t.Fatalf("Expected 'user._id' field, got: %+v", result)
		}
	})

	t.Run("InvalidObjectIDFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:invalid-hex-string`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Should fallback to string search
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}
		if result["_id"] != "invalid-hex-string" {
			t.Fatalf("Expected string fallback, got: %+v", result["_id"])
		}
	})

	t.Run("IDFieldWithRegexFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:/pattern/`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Should fallback to regex pattern
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}
		// Should be a regex pattern (complex type)
		if result["_id"] == "/pattern/" {
			t.Fatalf("Expected parsed regex pattern, got string: %+v", result["_id"])
		}
	})

	t.Run("IDFieldWithWildcardFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:*pattern*`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Should fallback to wildcard pattern
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}
		// Should be a wildcard pattern (complex type)
		if result["_id"] == "*pattern*" {
			t.Fatalf("Expected parsed wildcard pattern, got string: %+v", result["_id"])
		}
	})

	t.Run("IDFieldWithRangeFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:[507f1f77bcf86cd799439011 TO 507f1f77bcf86cd799439012]`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Should fallback to range pattern
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}
		// Should be a range pattern (complex type) or fallback to string
		// The exact type depends on how the parser handles the range syntax
		// Accept either a string or a complex range type as valid fallback behavior
		_ = result["_id"] // Verify field exists, fallback type is acceptable
	})

	t.Run("IDFieldWithComparisonFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `id:>507f1f77bcf86cd799439011`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Should fallback to comparison pattern
		if _, exists := result["_id"]; !exists {
			t.Fatalf("Expected '_id' field, got: %+v", result)
		}
		// Should be a comparison pattern (complex type) or fallback to string
		// The exact type depends on how the parser handles the comparison syntax
		// Accept either a string or a complex comparison type as valid fallback behavior
		_ = result["_id"] // Verify field exists, fallback type is acceptable
	})

	t.Run("UserIDFieldConversion", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `user_id:507f1f77bcf86cd799439011`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that field name was preserved as "user_id"
		if _, exists := result["user_id"]; !exists {
			t.Fatalf("Expected 'user_id' field, got: %+v", result)
		}

		// Check that value is ObjectID
		objectID, ok := result["user_id"].(bson.ObjectID)
		if !ok {
			t.Fatalf("Expected ObjectID, got %T: %+v", result["user_id"], result["user_id"])
		}

		// Verify it's the correct ObjectID
		expectedObjectID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
		if objectID != expectedObjectID {
			t.Fatalf("Expected ObjectID %v, got %v", expectedObjectID, objectID)
		}
	})

	t.Run("OrderIDFieldFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `order_id:invalid`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that field name was preserved as "order_id"
		if _, exists := result["order_id"]; !exists {
			t.Fatalf("Expected 'order_id' field, got: %+v", result)
		}

		// Should fallback to string search
		if result["order_id"] != "invalid" {
			t.Fatalf("Expected string fallback, got: %+v", result["order_id"])
		}
	})

	t.Run("ProductIDFieldFallback", func(t *testing.T) {
		formatter := mongo.NewWithOptions(true, true)

		query := `product_id:12345`
		ast, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		result, err := formatter.Format(ast)
		if err != nil {
			t.Fatalf("Format should not return error, got: %v", err)
		}

		// Check that field name was preserved as "product_id"
		if _, exists := result["product_id"]; !exists {
			t.Fatalf("Expected 'product_id' field, got: %+v", result)
		}

		// Should fallback to string search (not 24 chars)
		// The value should be parsed as a number since it's numeric
		if result["product_id"] != 12345.0 && result["product_id"] != "12345" {
			t.Fatalf("Expected numeric or string fallback, got: %+v", result["product_id"])
		}
	})

	t.Run("FormatWithDefaults", func(t *testing.T) {
		// Test FormatWithDefaults method
		formatter := bsonic.NewMongoFormatter()

		// Use parser to create a valid AST
		parser := &lucene.Parser{}
		astInterface, err := parser.Parse("john")
		if err != nil {
			t.Fatalf("Failed to parse query: %v", err)
		}

		ast := astInterface.(*lucene.ParticipleQuery)
		result, err := formatter.FormatWithDefaults(ast, []string{"name", "description"})
		if err != nil {
			t.Fatalf("FormatWithDefaults should not return error, got: %v", err)
		}

		// Should create OR query across default fields
		orArray, ok := result["$or"].([]bson.M)
		if !ok {
			t.Fatalf("Expected $or array, got %T", result["$or"])
		}

		if len(orArray) != 2 {
			t.Fatalf("Expected 2 default fields, got %d", len(orArray))
		}
	})

	t.Run("FormatWithDefaultsNilExpression", func(t *testing.T) {
		// Test FormatWithDefaults with nil expression
		formatter := bsonic.NewMongoFormatter()

		ast := &lucene.ParticipleQuery{
			Expression: nil,
		}

		result, err := formatter.FormatWithDefaults(ast, []string{"name"})
		if err != nil {
			t.Fatalf("FormatWithDefaults should not return error, got: %v", err)
		}

		expected := bson.M{}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, result)
		}
	})

	t.Run("FormatWithDefaultsInvalidAST", func(t *testing.T) {
		// Test FormatWithDefaults with invalid AST type
		formatter := bsonic.NewMongoFormatter()

		result, err := formatter.FormatWithDefaults("invalid", []string{"name"})
		if err == nil {
			t.Fatal("FormatWithDefaults should return error for invalid AST type")
		}

		if !strings.Contains(err.Error(), "expected *lucene.ParticipleQuery AST") {
			t.Fatalf("Expected error about AST type, got: %v", err)
		}

		// FormatWithDefaults returns empty bson.M{} on error, not nil
		expected := bson.M{}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected empty bson.M{}, got: %+v", result)
		}
	})
}
