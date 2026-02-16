# Go Vibe Starter

A starter pack for fast prototyping **production-ready** applications with Go.

This project is designed to be a ready foundation for building prototypes quickly with Claude Code. The idea is to invest enough time in a robust web server framework so you can focus on implementing business logic at speed.

---

## Why this stack?

### ✅ Authentication & Organizations
We use **Keycloak** for user management, including organizations/tenants. It’s common to have multiple organizations rather than a single tenant, and Keycloak is a good open-source solution I already know well.

### ✅ LLM-first development
Claude Code was chosen because it includes strong features for structured planning and tool use (including **Ralph Wiggum** and **skills**), which makes multi-step development reliable and predictable.

### ✅ No ORM
ORMs add unnecessary abstraction — for **humans** and **LLMs**. SQL is clear, explicit, and LLMs are great at writing it. That’s why this project uses **SQLC** to generate type-safe Go code directly from SQL queries.

### ✅ Lightweight REST with chi
We use **chi** for HTTP routing because:
- Lightweight (~1000 lines), minimal dependencies
- 100% compatible with `net/http` - idiomatic Go
- Great middleware ecosystem without magic
- REST is universally understood - easy integration with any service

Chi keeps routing simple and explicit while providing useful features like route grouping and path parameters.

### ✅ PostgreSQL
Postgres is reliable, flexible, and supports **pgvector**, which is great for embeddings and vector search. In the future, I may explore **Turso**.

---

## Goals

- Provide a production-grade base for rapid prototyping
- Use explicit, understandable infrastructure
- Minimize code and configuration overhead
- Make LLM-assisted development fast and practical

---

## Getting Started

> Documentation and setup instructions will be added soon.

---

## License

MIT
