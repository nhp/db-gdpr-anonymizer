package faker

import (
	"fmt"
	"strings"

	"github.com/go-faker/faker/v4"
)

// Generator generates fake data
type Generator struct{}

// NewGenerator creates a new faker generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates fake data based on the specified type
func (g *Generator) Generate(fakerType string) (string, error) {
	switch strings.ToLower(fakerType) {
	case "name":
		return faker.Name(), nil
	case "firstname":
		return faker.FirstName(), nil
	case "lastname":
		return faker.LastName(), nil
	case "email":
		return faker.Email(), nil
	case "phone", "phonenumber":
		return faker.Phonenumber(), nil
	case "address", "streetaddress":
		return "123 Main St", nil
	case "city":
		return "New York", nil
	case "state":
		return "NY", nil
	case "zipcode", "postcode":
		return "10001", nil
	case "country":
		return "USA", nil
	case "company":
		return "Acme Inc", nil
	case "jobtitle":
		return "Software Engineer", nil
	case "creditcard":
		return faker.CCNumber(), nil
	case "uuid":
		return faker.UUIDHyphenated(), nil
	case "ipv4":
		return faker.IPv4(), nil
	case "ipv6":
		return faker.IPv6(), nil
	case "url":
		return faker.URL(), nil
	case "username":
		return faker.Username(), nil
	case "password":
		return faker.Password(), nil
	case "numerify":
		return "123456789", nil
	case "sentence":
		return "This is a sample sentence.", nil
	case "paragraph":
		return "This is a sample paragraph. It contains multiple sentences. This is used for testing purposes.", nil
	default:
		return "", fmt.Errorf("unsupported faker type: %s", fakerType)
	}
}

// GetSupportedTypes returns a list of supported faker types
func (g *Generator) GetSupportedTypes() []string {
	return []string{
		"name",
		"firstname",
		"lastname",
		"email",
		"phone",
		"phonenumber",
		"address",
		"streetaddress",
		"city",
		"state",
		"zipcode",
		"postcode",
		"country",
		"company",
		"jobtitle",
		"creditcard",
		"uuid",
		"ipv4",
		"ipv6",
		"url",
		"username",
		"password",
		"numerify",
		"sentence",
		"paragraph",
	}
}

// GenerateSQL generates a SQL expression for the specified faker type
func (g *Generator) GenerateSQL(fakerType string, driver string) (string, error) {
	// For MySQL and PostgreSQL, we can use different approaches
	// In a real implementation, we would generate SQL that calls a function or uses a stored procedure
	// For simplicity, we'll just return a placeholder that will be replaced at runtime

	// In a real implementation, you might:
	// 1. For MySQL: Create a user-defined function that generates fake data
	// 2. For PostgreSQL: Use the pgcrypto extension for random data generation
	// 3. For both: Use prepared statements with parameters that are replaced with fake data at runtime

	return fmt.Sprintf("'[FAKER:%s]'", fakerType), nil
}

// ReplaceFakerPlaceholders replaces faker placeholders in SQL with actual fake data
func (g *Generator) ReplaceFakerPlaceholders(sql string) (string, error) {
	// Find all faker placeholders in the SQL
	// Format: '[FAKER:type:tableName:columnName:pkValue]'
	result := sql
	for {
		start := strings.Index(result, "'[FAKER:")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "]'")
		if end == -1 {
			return "", fmt.Errorf("invalid faker placeholder format in SQL: %s", sql)
		}
		end += start

		placeholder := result[start : end+2]
		fakerInfo := result[start+8 : end]

		// Extract the faker type from the placeholder
		// The format is now 'type:tableName:columnName:pkValue' or 'type:tableName:columnName'
		parts := strings.Split(fakerInfo, ":")
		fakerType := parts[0]

		// Generate fake data
		fakeData, err := g.Generate(fakerType)
		if err != nil {
			return "", err
		}

		// Replace only the first occurrence of the placeholder with fake data
		// This ensures each placeholder gets unique data even if they are of the same type
		result = strings.Replace(result, placeholder, fmt.Sprintf("'%s'", strings.ReplaceAll(fakeData, "'", "''")), 1)
	}

	return result, nil
}
