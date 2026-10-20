package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmptyName = errors.New("name cannot be empty or negative")
	ErrEmptyAge  = errors.New("age cannot be zero")
	ErrUnderage  = errors.New("person must be at least 18")
)

type Person struct {
	Name string `json:"name"`
	Age  uint8  `json:"age"` //Nobody lives past 255 nor can have a negative age so this is the most memory friendly data type for this :P
}

func main() {
	var person *Person
	// testString := `{"name":"Saleh", "age":20}`
	testWrongString := `{"name":"Saleh", "last_name":"Test", "age":20}` //Test unexpected non-desired fields
	person, err := fromJSON(testWrongString)
	if err != nil {
		fmt.Println("Could not parse person:", err)
		return
	}
	fmt.Println(*person)
}

func fromJSON(input string) (*Person, error) {
	var person Person

	inputString := bytes.NewReader([]byte(input))
	decoder := json.NewDecoder(inputString)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&person)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(input), &person); err != nil {
		return nil, fmt.Errorf("decode person JSON: %w", err)
	}

	//Note: JSON validation will throw an error if age is negative since I used uint8
	//Therefore, negative validation is not necessary here :P
	if strings.TrimSpace(person.Name) == "" {
		return nil, ErrEmptyName
	} else if person.Age == 0 {
		return nil, ErrEmptyAge
	} else if !isAdult(person) {
		return nil, ErrUnderage
	}

	return &person, nil

}

func isAdult(p Person) bool {
	return p.Age >= 18
}
