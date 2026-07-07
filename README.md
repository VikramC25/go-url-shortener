# Go URL Shortener

A lightweight URL shortening service built with **Go** and **PostgreSQL**, containerized with **Docker**, and deployed on **Render**. The application allows users to generate short URLs, track click counts, and redirect to the original destination while demonstrating production-oriented deployment practices.

## Live Demo

https://go-url-shortener-3fgq.onrender.com

---

## Features

- Shorten long URLs into unique, secure short links
- Redirect users to the original URL
- Track click analytics for each shortened URL
- Validate URLs before shortening
- Persistent PostgreSQL storage
- Automatic database schema initialization on startup
- Dockerized application with multi-stage builds
- Docker Compose support for local development
- Cloud deployment on Render

---

## Tech Stack

### Backend
- Go

### Database
- PostgreSQL

### DevOps
- Docker
- Docker Compose
- Render
- Multi-stage Docker builds

---

## Project Structure

```text
go-url-shortener/
│
├── data/
│   └── load.sql
│
├── internals/
│   └── models/
│       └── urls.go
│
├── static/
├── templates/
│
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .dockerignore
├── go.mod
├── go.sum
└── main.go
```

---

## Running Locally

### Prerequisites

- Docker Desktop
- Docker Compose

### Clone the repository

```bash
git clone https://github.com/VikramC25/go-url-shortener.git

cd go-url-shortener
```

### Configure environment variables

Create a `.env` file.

Example:

```env
DATABASE_URL=postgres://postgres:password@postgres:5432/urlshortener?sslmode=disable
SESSION_KEY=your-secret-key
PORT=8000
```

---

### Start the application

```bash
docker compose up --build
```

The application will be available at

```
http://localhost:8000
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| DATABASE_URL | PostgreSQL connection string |
| SESSION_KEY | Secret key used to sign session cookies |
| PORT | Port the application listens on |

---

## Docker

The project uses a **multi-stage Docker build** to produce a lightweight production image.

Build manually:

```bash
docker build -t go-url-shortener .
```

Run manually:

```bash
docker run -p 8000:8000 \
-e DATABASE_URL=<database-url> \
-e SESSION_KEY=<secret> \
go-url-shortener
```

---

## Docker Compose

Docker Compose provisions the complete local development environment, including:

- Go application
- PostgreSQL database
- Docker networking
- Persistent database volume

Run:

```bash
docker compose up --build
```

---

## Deployment

The application is deployed on **Render** using Docker.

Deployment includes:

- Docker container
- Managed PostgreSQL database
- Environment variable configuration
- Automatic schema initialization on startup

---

## API Routes

| Method | Endpoint | Description |
|---------|----------|-------------|
| GET | `/` | Homepage |
| POST | `/` | Create a shortened URL |
| GET | `/o/:url` | Redirect to the original URL |

---

## Future Improvements

- User authentication
- Custom aliases
- URL expiration
- Rate limiting
- QR code generation
- REST API endpoints
- GitHub Actions CI/CD pipeline
- Prometheus and Grafana monitoring

---
