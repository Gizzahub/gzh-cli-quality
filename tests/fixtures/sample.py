"""Sample Python module for testing."""


def sample_function(input_str: str) -> str:
    """Sample function that processes a string.

    Args:
        input_str: Input string to process

    Returns:
        Processed string
    """
    if not input_str:
        return "empty"
    return f"value: {input_str}"


class SampleClass:
    """Sample class for testing."""

    def __init__(self, name: str, value: int):
        """Initialize SampleClass.

        Args:
            name: Name value
            value: Integer value
        """
        self.name = name
        self.value = value

    def get_description(self) -> str:
        """Get description.

        Returns:
            Description string
        """
        return f"{self.name}: {self.value}"
