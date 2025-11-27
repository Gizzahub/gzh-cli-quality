// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//! Sample Rust module for testing.

/// Sample function that processes a string.
pub fn sample_function(input: &str) -> String {
    if input.is_empty() {
        return "empty".to_string();
    }
    format!("value: {}", input)
}

/// Sample struct for testing.
pub struct SampleStruct {
    pub name: String,
    pub value: i32,
}

impl SampleStruct {
    /// Create a new SampleStruct.
    pub fn new(name: String, value: i32) -> Self {
        Self { name, value }
    }

    /// Get description.
    pub fn get_description(&self) -> String {
        format!("{}: {}", self.name, self.value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sample_function() {
        assert_eq!(sample_function("test"), "value: test");
        assert_eq!(sample_function(""), "empty");
    }

    #[test]
    fn test_sample_struct() {
        let sample = SampleStruct::new("test".to_string(), 42);
        assert_eq!(sample.get_description(), "test: 42");
    }
}
