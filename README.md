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

### ✅ RPC-first architecture
We use **Connect RPC**, which gives you:
- a simple way to build web servers
- the benefits of gRPC
- protobufs as the source of truth

This is especially useful for microservices or systems with multiple services communicating together. Even if OpenAPI/REST is common, this project focuses on reducing boilerplate and keeping mappings minimal.

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
