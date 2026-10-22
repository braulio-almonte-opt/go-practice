## Task 2: REST API with Gin (Production-Ready Architecture)

Build a production-ready REST API using the following stack:
* **Framework:** [Gin](https://github.com/gin-gonic/gin)
* **Authentication:** JWT (JSON Web Tokens)
* **Database:** MongoDB
* **Dependency Injection:** [Google Wire](https://github.com/google/wire)
* **Mocking:** [Mockery](https://github.com/vektra/mockery)

---

### Endpoints

These are the endpoints the API is expected to provide. Protected endpoints
require a valid JWT sent in the `Authorization` header using the Bearer scheme.

#### Authentication

| Method and path | Purpose | Access |
|---|---|---|
| `POST /login` | Authenticate with login credentials. On success, return a JWT for use on protected endpoints. | Public |
| `POST /logout` | Log out on the client by discarding the token. The token remains valid until it expires; immediate server-side revocation is not currently required. | Valid JWT |

#### User Management

| Method and path | Purpose | Access |
|---|---|---|
| `POST /users` | Register a new user. The request body contains the user fields accepted for registration. | Public for now |
| `GET /users/me` | Return the profile of the user identified by the JWT. The client does not supply a user ID; the server gets the identity from the token. | Valid JWT |
| `GET /users/:id` | Return the user whose ID is in the URL. A user may request their own ID; requesting another user's ID requires the `admin` role. | Valid JWT; owner or admin |
| `GET /users` | Return the user list. | Valid JWT; admin only |

---

### Functional Requirements

1. **Authentication:**
   * `GET /users/me`, `GET /users/:id`, and `GET /users` must require a valid JWT token.

2. **Authorization / RBAC:**
   * For `GET /users/:id`, allow access when the authenticated user's ID
     matches the requested ID, or when the authenticated user has an `"admin"`
     role.
   * For `GET /users`, allow access only when the authenticated user has an
     `"admin"` role.

---

### 🔧 Architecture Requirements

Structure the project following a standard multi-layer pattern:
```
Controller (Handler) ──> Service (Business Logic) ──> DAO / Repository (Data Access)
```

* **Dependency Injection:** Use Google Wire to wire up controllers, services, and DAOs.
* **Separation of Concerns:** Business logic must reside strictly in the Service layer.
* **Interface-Based DAO:** The DAO/Repository layer must be interface-driven to support mocking.
* **Database:** MongoDB. MySQL integration is an optional bonus after the MongoDB API is working.

---

### Testing Requirements

* **Unit Tests:** Write comprehensive unit tests for:
  * Service layer
  * DAO / Repository layer
* **Mocking:** Use Mockery to generate mocks for DAO interfaces.
* **Test Coverage:**
  * Success cases
  * Failure cases / validation errors
  * Unauthorized / Forbidden access cases
* **Isolation Rule:** No direct database calls inside service-layer unit tests.

### Regenerating Repository Mocks

Repository mocks are generated from `repository.UserRepository` with Mockery
v2.53.8. After changing the interface, run this command from the project root:

```sh
go run github.com/vektra/mockery/v2@v2.53.8
```

The generated mock is saved in `repository/mocks/user_repository.go`. The
version is pinned in the command so regeneration uses the same Mockery release.

---

### Bonus Challenges

* [ ] **Request Validation:** Add middleware for automatic request payload validation.
* [ ] **Password Security:** Implement password hashing using `bcrypt`.
* [ ] **Pagination:** Add page and limit parameters to `GET /users`.
* [ ] **Structured Logging:** Integrate structured logging (e.g., `zap` or `logrus`).
* [ ] **Graceful Shutdown:** Implement graceful server shutdown on OS signals (e.g., `SIGTERM`, `SIGINT`).
* [ ] **MySQL Integration:** Integrate the same logic with MySQL after having succesfully used MongoDB