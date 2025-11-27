// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package fixtures

// SampleFunction demonstrates a simple Go function.
func SampleFunction(input string) string {
	if input == "" {
		return "empty"
	}
	return "value: " + input
}

// SampleStruct demonstrates a simple struct.
type SampleStruct struct {
	Name  string
	Value int
}

// NewSampleStruct creates a new SampleStruct.
func NewSampleStruct(name string, value int) *SampleStruct {
	return &SampleStruct{
		Name:  name,
		Value: value,
	}
}
