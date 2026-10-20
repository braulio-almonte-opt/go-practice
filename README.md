# Task 1: JSON Parser and Validator

Create a Go program that:

**Requirements**
* Define a struct
```
type Person struct {
    Name: string `json:"name"`
    Age: int `json:"age"`
  }
```

* Implement
  * ```fromJSON(input string) (*Person, error)```
    * Validate that:
      1. Name is not empty
      2. Age is not negative
    * Return meaningful errors
    * Reject unknown JSON fields
  * ```isAdult(p Person)```
    * Returns true if ```age >= 18```
  * Use this JSON to test:
    * {"name": "Saleh", "age": 20}
