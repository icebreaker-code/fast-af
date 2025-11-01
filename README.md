# Fast-AF: Getting Started for Development

Welcome to the Fast-AF project! This guide will help you set up your development environment and get started quickly.

## Prerequisites
- [Go](https://golang.org/dl/) (version 1.18 or higher recommended)
- [MongoDB](https://www.mongodb.com/try/download/community) (for local development)

## Setup Instructions

1. **Clone the repository**
   ```sh
git clone https://github.com/icebreaker-code/fast-af.git
cd fast-af
```

2. **Install dependencies**
   ```sh
go mod tidy
```

3. **Configure environment variables**
   - Copy the sample config file if available, or set up your own in `config/configs.go`.
   - Ensure MongoDB connection details are correct.

4. **Run MongoDB locally**
   - Start your MongoDB server (default port: 27017).

5. **Start the application**
   ```sh
go run cmd/main.go
```

6. **Access the API**
   - The server will start on the default port (e.g., `localhost:8080`).
   - Use tools like [Postman](https://www.postman.com/) or [curl](https://curl.se/) to interact with the API endpoints.

## Project Structure
- `cmd/` - Entry point for the application
- `config/` - Configuration files
- `controllers/` - API controllers
- `database/` - Database connection logic
- `models/` - Data models
- `routes/` - API route definitions

## Useful Commands
- Run tests:
  ```sh
go test ./...
```
- Build the project:
  ```sh
go build -o fast-af cmd/main.go
```

## Contributing
Feel free to open issues or submit pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the MIT License.
