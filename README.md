## Project Description

A Go + Gin web service that generates personalized stories based on user preferences, using a **worker pool architecture, Redis for OTP, and PostgreSQL**.    

This project demonstrates a backend structure featuring **Clean architecture, rate limiting, structured logging, JWT authentication, config management, health checks, and background job processing**.  

## Core Functionality

Generates custom stories using Gemini AI based on user preferences.
Implements a two-stage job pipeline:  
  * Story Generation Job → sent to a channel.

  * Email Notification Job → processed by separate workers and sent via SMTP server.

Uses worker pool pattern and Go channels for concurrency management.

## Tech Stack

Language: Go
Framework: Gin  
Database: PostgreSQL  
Cache / OTP Store: Redis  
Migrations: Goose  
Configuration: Viper  
Rate Limiting: Leaky Bucket Algorithm  
Authentication: JWT Middleware  
Logging: Slog  
Email Service: Mailtrap SMTP  
Health Checks: /health with runtime stats and pprof  

## Authentication & Validation

JWT Middleware — validates user tokens on protected routes.  
Request Validation — validates JSON payloads and user preferences.  
Password Rules — enforced using validator package and middleware.  
OTP Flow — Redis stores and validates OTP codes for login/registration.  

## Rate Limiting
Implements Leaky Bucket algorithm to prevent abuse.  

## health endpoint exposes:
Application status  
Runtime statistics  
Integrated pprof profiling for performance insights  

## Middleware
JWT Auth: Secure endpoints  
JSON Validation: Validate incoming requests  
Input Validator: Enforce user input rules  
Logger: Structured logging with slog  
Panic Recovery: Gracefully handle crashes  
Rate Limiter: Apply per-user limits   
