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

### Logging In

`POST /login` accepts JSON containing `username` and `password`. A successful
login returns `200` with an HS256-signed bearer token and its expiration:

```json
{
  "access_token": "<signed JWT>",
  "token_type": "Bearer",
  "expires_at": "2026-10-22T11:00:00Z"
}
```

Tokens expire one hour after issue and contain the user's MongoDB ID as `sub`
and the user's role as `role`. Configure `JWT_SECRET` with at least 32 random
bytes before starting the application, for example:

```sh
export JWT_SECRET="$(openssl rand -base64 32)"
```

The JWT middleware verifies the Bearer scheme, HS256 signature, expiration,
and user identity claims. It protects `GET /users/me`, `GET /users/:id`, and
`GET /users`; login and registration remain public. On successful verification,
it makes the subject and role available to Gin handlers under
`authenticatedUserID` and `authenticatedRole`. Owner/admin authorization is
enforced under AUTHZ1 and AUTHZ2.

Unknown usernames and incorrect passwords receive the same `401` response.
Missing or malformed credentials receive `400`; repository or signing failures
receive `500`. Password hashing remains the separate BONUS2 task; until it is
implemented, login compares passwords in the same format currently stored by
registration. Tokens are issued by API6 and verified by AUTH1 middleware on
protected routes.

Public registration always assigns the `user` role, even if a request body
includes a different role. Assign the `admin` role only through a trusted
administrative process; otherwise anyone could grant themselves access to the
user list or other accounts.

### Testing Owner and Admin Access in HTTPie

The automated authorization tests run with `go test ./...`. To manually test
the role behavior in HTTPie GUI, use a local development database only:

1. Register two accounts using `POST http://localhost:8080/users`, each with a
   JSON body containing `username` and `password`. Registration assigns both
   accounts the `user` role.
2. Promote one test account to admin in the local MongoDB database using
   `mongosh`:

   ```javascript
   use goAPI
   db.users.updateOne(
     { username: "test-admin" },
     { $set: { role: "admin" } }
   )
   ```

   Do this only in a local test database. Public registration intentionally
   cannot grant admin privileges.
3. In HTTPie GUI, send `POST http://localhost:8080/login` with JSON containing
   the account's username and password. Copy `access_token` from the response.
   Log in again after promoting the admin account so its token contains the
   updated role.
4. To test as the regular user, set **Auth** to **Bearer** and paste that
   user's token. Send:
   - `GET http://localhost:8080/users/<regular-user-id>` — expect `200` (owner).
   - `GET http://localhost:8080/users/<other-user-id>` — expect `403`.
   - `GET http://localhost:8080/users` — expect `403`.
   - `GET http://localhost:8080/users/me` — expect `200`.
5. Replace the bearer token with the admin token and repeat:
   - `GET http://localhost:8080/users/<other-user-id>` — expect `200`.
   - `GET http://localhost:8080/users` — expect `200`.

You can retrieve the generated IDs from the admin list response or inspect the
test records in MongoDB. Missing or invalid tokens still receive `401`.

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

### Running MongoDB Repository Integration Tests

Repository integration tests require an explicitly configured MongoDB test
server. Set `GO_PRACTICE_TEST_MONGODB_URI` and run:

```sh
GO_PRACTICE_TEST_MONGODB_URI='mongodb://localhost:27017' go test ./repository -v
```

The CRUD integration test creates a uniquely named `go_practice_test_*`
database, uses only its own `users` collection, and drops that exact database
during cleanup. Use a test MongoDB deployment; the integration test user needs
permission to create and drop databases. If the environment variable is unset,
the CRUD integration test is skipped. Disconnected-client error tests run
without a MongoDB server.

### Regenerating Dependency Injection

Google Wire connects the MongoDB repository, user service, and Gin router.
After changing providers or constructor dependencies, regenerate the checked-in
injector from the project root:

```sh
go run github.com/google/wire/cmd/wire@v0.7.0 gen ./...
```

The generated `wire_gen.go` is used by normal builds. MongoDB connection setup
and cleanup remain in `main.go`; the initialized client is passed into the
generated injector.

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