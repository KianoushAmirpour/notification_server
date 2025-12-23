## Project Description 

This repository contains a web service for generating and delivering personalized stories on demand. The project is primarily focused on following **Clean Architecture principles** and common backend best practices.  

The service exposes APIs for user management and story requests, while offloading heavy or slow operations to background workers.  

## High-Level Overview

API layer handles authentication, validation, and request orchestration  

Long-running tasks (story generation, email sending) are processed asynchronously  

Strong focus on security, observability  


## Key Features & Architecture
### Asynchronous Background Processing

Time-consuming tasks such as **Story generation and Email notifications** are not handled inline with API requests.

Instead, they are:

* Published to **Redis Streams**

* Processed by **worker pool** as consumers

* Retried using a **backoff strategy**

* Sent to a **Dead Letter Queue (DLQ)** after exceeding retry limits

This prevents API blocking and keeps request latency low.

### Security & Access Control
#### User Verification (OTP)

* User registration is protected using **One-Time Passwords (OTP)**

* OTPs are stored in **Redis**

* Only the **hashed OTP** is persisted to improve security

#### JWT Authentication

* Secured using **JWT-based authentication**

* Both **access tokens and refresh tokens** are implemented

* Refresh tokens:

  * Are stored in the database

  * Support rotation and revocation

  * Are validated on every token refresh request

This setup allows proper session control and logout handling.

### Rate Limiting

* Uses the **Token Bucket algorithm**

* Enforced per IP

* Implemented with:

  * Redis

  * Lua scripts for atomic rate-limit checks

This protects the service from abuse and excessive traffic.

### Input Validation

Requests are strictly validated using dedicated middlewares, including:

* Content-Type checks

* Request body structure validation

* Password rules

* User preference validation

Invalid requests are rejected early in the request lifecycle.

### Observability & Logging

Uses structured logging with slog

Logs follow **ECS (Elastic Common Schema)** conventions

Designed to integrate cleanly with **ElasticSearch**

This makes it easier to trace requests, debug issues, and monitor system behavior in production.

## Tech Stack
Language: Go
Framework: Gin  
Database: PostgreSQL  
OTP Store: Redis  
Migrations: goose  
Configuration: Viper  
Rate Limiting: Token Bucket Algorithm and Redis
Authentication: JWT Middleware  
Logging: Slog  
Email Service: Mailtrap SMTP  
Health Checks: /health with runtime stats