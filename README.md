# DB GDPR Anonymizer

A Go-based tool for anonymizing database content to comply with GDPR and other data protection regulations. This tool helps you replace sensitive personal data in your databases with realistic fake data while maintaining referential integrity.

## Features

- Anonymize specific tables and columns in your database
- Support for multiple database drivers (MySQL currently implemented)
- Various anonymization strategies (fake data generation, nullification, custom values)
- Dry-run mode to preview changes without modifying the database
- Parallel processing for improved performance
- Detailed reporting in text or JSON format
- Comprehensive logging

## Installation

### Prerequisites

- Go 1.16 or higher
- Access to a database you want to anonymize

### Building from source

```bash
# Clone the repository
git clone https://github.com/yourusername/db-gdpr-anonymizer.git
cd db-gdpr-anonymizer

# Build the binary
go build -o anonymize-db ./cmd/anonymize-db
```

### Static Linking

To create a statically linked binary that includes glibc, you can use the following build flags:

```bash
# Statically link all dependencies including glibc
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o anonymize-db ./cmd/anonymize-db
```

This creates a fully static binary that can be deployed to any Linux system regardless of the installed glibc version.

### Cross-Building for Different Platforms

Go makes it easy to cross-compile for different operating systems and architectures using the `GOOS` and `GOARCH` environment variables:

#### For Linux (various architectures)

```bash
# For Linux AMD64 (64-bit)
GOOS=linux GOARCH=amd64 go build -o anonymize-db-linux-amd64 ./cmd/anonymize-db

# For Linux ARM64 (e.g., Raspberry Pi 4)
GOOS=linux GOARCH=arm64 go build -o anonymize-db-linux-arm64 ./cmd/anonymize-db

# For Linux ARM (e.g., older Raspberry Pi)
GOOS=linux GOARCH=arm go build -o anonymize-db-linux-arm ./cmd/anonymize-db
```

#### For macOS

```bash
# For macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o anonymize-db-darwin-amd64 ./cmd/anonymize-db

# For macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o anonymize-db-darwin-arm64 ./cmd/anonymize-db
```

#### For Windows

```bash
# For Windows AMD64 (64-bit)
GOOS=windows GOARCH=amd64 go build -o anonymize-db-windows-amd64.exe ./cmd/anonymize-db

# For Windows 386 (32-bit)
GOOS=windows GOARCH=386 go build -o anonymize-db-windows-386.exe ./cmd/anonymize-db
```

#### Combined Static Linking and Cross-Building

You can combine static linking with cross-building:

```bash
# Static build for Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o anonymize-db-linux-amd64-static ./cmd/anonymize-db
```

Note: When `CGO_ENABLED=0`, Go will automatically create a statically linked binary. This is the simplest approach for cross-platform builds, but it may not work if your application depends on C libraries.

## Usage

```bash
./anonymize-db --config=your-config.yaml [options]
```

### Command-line options

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to YAML configuration file | (required) |
| `--dry-run` | Run in simulation mode without making changes | false |
| `--report` | Final report format (json or text) | text |
| `--log` | Directory for log files | logs |
| `--workers` | Number of parallel workers | Number of CPU cores |

## Configuration

The configuration file is in YAML format and specifies:
1. Database connection details
2. Tables and columns to anonymize
3. Anonymization strategies for each column

### Example Configuration

```yaml
database:
  host: localhost
  port: 3306
  user: dbuser
  password: dbpassword
  name: mydatabase
  driver: mysql

tables:
  customers:
    primary_key: "id"  # Optional: specify the primary key column (defaults to "id")
    columns:
      email:
        type: faker.email
      first_name:
        type: faker.firstname
      last_name:
        type: faker.lastname
      phone:
        type: faker.phone
      address:
        type: faker.address

  orders:
    primary_key: "order_id"  # Use this if the primary key is not "id"
    columns:
      customer_email:
        type: faker.email
      shipping_address:
        null: true
      credit_card_number:
        value: "XXXX-XXXX-XXXX-1234"
```

### Anonymization Strategies

The tool supports three main anonymization strategies:

1. **Faker Types**: Replace with realistic fake data
   ```yaml
   email:
     type: faker.email
   ```

2. **Nullification**: Set the field to NULL
   ```yaml
   shipping_address:
     null: true
   ```

3. **Custom Values**: Set a specific value
   ```yaml
   credit_card_number:
     value: "XXXX-XXXX-XXXX-1234"
   ```

### Available Faker Types

| Type | Description | Example |
|------|-------------|---------|
| name | Full name | John Smith |
| firstname | First name | John |
| lastname | Last name | Smith |
| email | Email address | john.smith@example.com |
| phone, phonenumber | Phone number | 555-123-4567 |
| address, streetaddress | Street address | 123 Main St |
| city | City name | New York |
| state | State/province | California |
| zipcode, postcode | Postal code | 10001 |
| country | Country name | United States |
| company | Company name | Acme Inc |
| jobtitle | Job title | Software Engineer |
| creditcard | Credit card number | 4111111111111111 |
| uuid | UUID | 550e8400-e29b-41d4-a716-446655440000 |
| ipv4 | IPv4 address | 192.168.1.1 |
| ipv6 | IPv6 address | 2001:0db8:85a3:0000:0000:8a2e:0370:7334 |
| url | URL | http://example.com |
| username | Username | jsmith |
| password | Password | p@ssw0rd |
| numerify | Random numeric string | 123456789 |
| sentence | Random sentence | This is a sample sentence. |
| paragraph | Random paragraph | This is a sample paragraph. It contains multiple sentences. |

## Example Reports

### Text Report
```
Anonymization Report
====================
Date: 2023-05-15 14:30:45

Summary:
- Tables processed: 2
- Fields processed: 8
- Rows scanned: 1250
- Rows affected: 1250
- Duration: 3.45s

Details:
Table: customers
  - Field: email (faker.email)
    * Rows scanned: 1000
    * Rows affected: 1000
  - Field: first_name (faker.firstname)
    * Rows scanned: 1000
    * Rows affected: 1000
  ...
```

### JSON Report
```json
{
  "summary": {
    "date": "2023-05-15T14:30:45Z",
    "totalTables": 2,
    "totalFields": 8,
    "totalRowsScanned": 1250,
    "totalRowsAffected": 1250,
    "duration": "3.45s"
  },
  "details": [
    {
      "tableName": "customers",
      "fieldName": "email",
      "strategy": "faker.email",
      "rowsScanned": 1000,
      "rowsAffected": 1000,
      "duration": "1.2s",
      "error": null
    }
  ]
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
