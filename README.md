# Choysum

English | [简体中文](README_zh-cn.md)

🌱 **Choysum: An efficient TypeScript runtime custom-built for customization and extension, featuring out-of-the-box ERP core modules.**

*   **⚡ Full-Stack TypeScript Development**: Unified TypeScript for both frontend and backend. Lower development barrier, easier team collaboration.
*   **🧩 Modular Customization**: Compile-time model-based extension architecture allows free overriding and extension of native features, with perfect isolation between core code and custom business logic.
*   **🚀 Go-Powered Extreme Performance**: Zero binary dependencies, single-file deployment, out-of-the-box ready. Business logic runs directly on the built-in high-performance TypeScript engine, no external Node.js required.

---

## ✨ Core Features

### 1. Compile-Time Model-Based Extension Architecture
Designed for complex customization needs, Choysum draws inspiration from iterative modular extension principles (inspired by **Odoo**). To eliminate performance overhead caused by "runtime magic" in dynamic languages, we have moved **all model and page merging and replacement operations to compile time**. This not only achieves a clean modular design that keeps core code pristine, but also delivers zero runtime overhead, significantly improving system maintainability and seamless upgrade efficiency.

### 2. TypeScript-First Full-Stack Development Experience
All business modules (including backend data models, service interfaces, and frontend **Vue** views) are written in strongly-typed **TypeScript**. Developers only need to master **TS/Vue** to build full-stack applications from database schema design to page rendering. **End-to-end type safety** not only drastically reduces the learning curve for teams, but also catches most common type errors at compile time.

### 3. Native gRPC Protocol Support
The network layer is built on the gRPC-Native philosophy. Backend TypeScript domain services are automatically mapped to standard gRPC interfaces, and the frontend communicates with the backend via the framework's built-in grpc-web protocol. This eliminates boilerplate code for manual routing and serialization. Additionally, the system automatically generates standardized `.proto` contracts, allowing you to easily create cross-language SDKs and seamlessly integrate with existing microservice architectures.

### 4. High-Performance Go Core Engine
The core engine is built in **Go**, with an embedded TypeScript compilation and execution engine. Business-layer TypeScript code **runs entirely without external Node.js dependencies**, executing efficiently in isolated runtime instances created by Go. This leverages Go's excellent concurrent scheduling to ensure system stability, while deployment only requires a single compiled Go executable, completely eliminating complex runtime environment dependencies.

### 5. Multi-Database Dialect Support
The framework provides a high-level abstraction and unified dialect compatibility layer at the underlying ORM level. It natively supports the three major relational databases: **PostgreSQL, MySQL, and SQLite** (more databases are planned). This comprehensive low-level adaptation frees developers from worrying about database-specific syntax differences, allowing them to flexibly choose or migrate underlying storage based on business scale and deployment scenarios.