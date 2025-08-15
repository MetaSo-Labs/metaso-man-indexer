# Group Chat Module Swagger Documentation Management

## Overview

This directory contains the Swagger documentation management tools for the group chat module. All swagger-related functionality is centralized here.

## Directory Structure

```
api/swagger/
├── swagger.go      # Swagger route configuration
├── generate.go     # Documentation generation tool
├── cli.go          # Command line interface
├── cmd/
│   └── main.go     # CLI tool entry point
├── generate.sh     # Generation script
└── README.md       # This file
```

## Usage

### 1. Generate Swagger Documentation

#### Method 1: Using script (recommended)
```bash
cd api/swagger
./generate.sh
```

#### Method 2: Using Go tool
```bash
go run api/swagger/cmd/main.go generate
```

#### Method 3: Directly using swag command
```bash
swag init -g cmd/main.go -o ./docs --parseDependency --parseInternal
```

### 2. Clean Documentation
```bash
go run api/swagger/cmd/main.go clean
```

### 3. Regenerate Documentation
```bash
go run api/swagger/cmd/main.go regenerate
```

### 4. Install swag tool
```bash
go run api/swagger/cmd/main.go install
```

## Start Server

After generating documentation, start the server:

```bash
go run cmd/main.go -host 0.0.0.0 -port 7568
```

## Access Swagger Documentation

After the server starts, visit the following address to view API documentation:

```
http://0.0.0.0:7568/group-chat/docs/index.html
```

## Development Workflow

### 1. After Modifying API Code

1. Update API code and comments
2. Regenerate documentation:
   ```bash
   cd api/swagger && ./generate.sh
   ```
3. Restart server
4. Visit documentation to see updates

### 2. Adding New APIs

1. Add new API function in `api/db_controller.go`
2. Add complete Swagger comments
3. Register route in `api/db_routes.go`
4. Generate documentation: `cd api/swagger && ./generate.sh`
5. Start server for testing

## Swagger Comment Format

```go
// @Summary API title
// @Description API detailed description
// @Tags Database Query
// @Accept json
// @Produce json
// @Param parameter_name query parameter_type required "parameter description"
// @Success 200 {object} map[string]interface{} "success description"
// @Failure 400 {object} map[string]interface{} "error description"
// @Router /api/path [get]
func YourAPIHandler(c *gin.Context) {
    // Your code
}
```

## Notes

1. **Comment Position**: Swagger comments must be directly above the function
2. **Comment Format**: Must use comments starting with `//`
3. **Parameter Types**: Ensure parameter types match the actual code
4. **Route Path**: Ensure the path in `@Router` matches the actual route
5. **Tag Usage**: Use appropriate tags to organize APIs

## Troubleshooting

### 1. swag command not found
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Generated documentation incomplete
Check if comment format is correct, ensure:
- Comments start with `//`
- Each comment is above the function
- Parameter types and descriptions are correct

### 3. Swagger UI inaccessible
Ensure:
- Server is running
- Access correct URL: `http://0.0.0.0:7568/group-chat/swagger/index.html`
- Port is not occupied by other services

## File Description

- `swagger.go`: Configure Swagger routes and URL generation
- `generate.go`: Provide documentation generation, cleanup, regeneration and other functions
- `cli.go`: Command line interface, supports various operations
- `generate.sh`: Simple generation script for quick use

## More Resources

- [Swaggo Official Documentation](https://github.com/swaggo/swag)
- [Swagger Specification](https://swagger.io/specification/)
- [Gin Framework Documentation](https://gin-gonic.com/docs/) 