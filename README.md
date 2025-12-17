## Project Description

This project is a web service designed to generate and deliver personalized stories on demand. It showcases a robust, maintainable architecture following **Clean Architecture** principles.

Key Features and Architecture
Background Processing: Utilizes a **worker pool pattern** to efficiently offload time-consuming tasks, including story generation and email notifications, preventing API blocking.

Security & Access Control: Implements secure JWT authentication **(including access token and refresh token with rotation and revocation), One-Time Password (OTP) validation, and API rate limiting.**

Detailed Functionality
Asynchronous Task Handling: Time-consuming operations (story generation and email notification) are offloaded to channels and processed efficiently using the worker pool pattern.

Secure User Verification: User registration is validated with One-Time Passwords (OTP). Redis is used to store OTP codes, and for enhanced security, the hashed version of the OTP is saved.

JWT Security Implementation: API endpoints are secured with JWTs. Both access and refresh tokens are implemented. Refresh tokens are persisted in the database, and a robust verification process is performed every time a user requests new tokens.

Rate Limiting: The Token Bucket algorithm has been utilized for effective rate limiting enforced per IP address.

Input Validation: User requests and inputs are strictly validated through dedicated middlewares. This includes checks for Content-Type, request body structure, passwords, and user preferences.

Observability: Structured logging (using Go's slog) is implemented to effectively track and monitor request flow and processing throughout the service.
  
<!-- 
A Go + Gin web service that generates personalized stories based on user preferences, using a **worker pool architecture, Redis for OTP, and PostgreSQL**.    

This project demonstrates a backend structure featuring **Clean architecture, rate limiting, structured logging, JWT authentication, config management, health checks, and background job processing**.   -->

<!-- ## Core Functionality

Generates custom stories using Gemini AI based on user preferences.
Implements a two-stage job pipeline:  
  * Story Generation Job → sent to a channel.

  * Email Notification Job → processed by separate workers and sent via SMTP server.

Uses worker pool pattern and Go channels for concurrency management. -->

## Tech Stack

Language: Go
Framework: Gin  
Database: PostgreSQL  
OTP Store: Redis  
Migrations: Goose  
Configuration: Viper  
Rate Limiting: Token Bucket Algorithm  
Authentication: JWT Middleware  
Logging: Slog  
Email Service: Mailtrap SMTP  
Health Checks: /health with runtime stats and pprof  

<!-- ## Authentication & Validation

JWT Middleware — validates user tokens on protected routes.  
Request Validation — validates JSON payloads and user preferences.  
Password Rules — enforced using validator package and middleware.  
OTP Flow — Redis stores and validates OTP codes for login/registration.   -->

<!-- ## Rate Limiting
Implements Leaky Bucket algorithm to prevent abuse.   -->

<!-- ## health endpoint exposes:
Application status  
Runtime statistics  
Integrated pprof profiling for performance insights  

## Middleware
JWT Auth: Secure endpoints  
JSON Validation: Validate incoming requests  
Input Validator: Enforce user input rules  
Logger: Structured logging with slog  
Panic Recovery: Gracefully handle crashes  
Rate Limiter: Apply per-user limits    -->
