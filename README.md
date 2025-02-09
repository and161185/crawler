# Crawler

Crawler is a web crawler written in Go that allows users to extract information from web pages efficiently. The project includes basic functionality for fetching and parsing HTML content and can be extended for more advanced crawling tasks.

## Features

- Fetches HTML content from web pages.
- Extracts links and metadata.
- Supports concurrency for faster crawling.
- Includes CI/CD configuration for automated builds.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/and161185/crawler.git
   cd crawler
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the application:
   ```bash
   go build -o crawler .
   ```

## Usage

Run the crawler with a target URL:
```bash
./crawler -url https://example.com
```

### Command-line Flags
- `-url`: Specifies the target URL.
- `-depth`: Sets the crawling depth (default: 1).
- `-concurrency`: Defines the number of concurrent requests (default: 10).

## CI/CD

The repository includes a GitHub Actions workflow located in `.github/workflows/`. This workflow automatically builds the application on push events.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

