// Package helpers provides shared test utilities for BSON comparison and testing.
package helpers

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// CompareBSONValues compares BSON values for testing
func CompareBSONValues(actual, expected interface{}) bool {
	// Handle time.Time comparison
	if actualTime, ok := actual.(time.Time); ok {
		return compareTimeValues(actualTime, expected)
	}

	// Handle bson.M comparison
	if actualMap, ok := actual.(bson.M); ok {
		return compareBSONMaps(actualMap, expected)
	}

	// Handle []bson.M comparison
	if actualArray, ok := actual.([]bson.M); ok {
		return compareBSONArrays(actualArray, expected)
	}

	// Default comparison
	return actual == expected
}

// compareTimeValues compares time.Time values
func compareTimeValues(actualTime time.Time, expected interface{}) bool {
	expectedTime, ok := expected.(time.Time)
	return ok && actualTime.Equal(expectedTime)
}

// compareBSONMaps compares bson.M values
func compareBSONMaps(actualMap bson.M, expected interface{}) bool {
	expectedMap, ok := expected.(bson.M)
	if !ok {
		return false
	}

	if len(actualMap) != len(expectedMap) {
		return false
	}

	for key, expectedValue := range expectedMap {
		actualValue, exists := actualMap[key]
		if !exists || !CompareBSONValues(actualValue, expectedValue) {
			return false
		}
	}
	return true
}

// compareBSONArrays compares []bson.M values
func compareBSONArrays(actualArray []bson.M, expected interface{}) bool {
	expectedArray, ok := expected.([]bson.M)
	if !ok {
		return false
	}

	if len(actualArray) != len(expectedArray) {
		return false
	}

	for i, expectedValue := range expectedArray {
		if !CompareBSONValues(actualArray[i], expectedValue) {
			return false
		}
	}
	return true
}
