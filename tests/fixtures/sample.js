/**
 * Sample JavaScript module for testing.
 */

/**
 * Sample function that processes a string.
 * @param {string} inputStr - Input string to process
 * @returns {string} Processed string
 */
function sampleFunction(inputStr) {
  if (!inputStr) {
    return 'empty';
  }
  return `value: ${inputStr}`;
}

/**
 * Sample class for testing.
 */
class SampleClass {
  /**
   * Create a SampleClass.
   * @param {string} name - Name value
   * @param {number} value - Integer value
   */
  constructor(name, value) {
    this.name = name;
    this.value = value;
  }

  /**
   * Get description.
   * @returns {string} Description string
   */
  getDescription() {
    return `${this.name}: ${this.value}`;
  }
}

module.exports = {
  sampleFunction,
  SampleClass,
};
