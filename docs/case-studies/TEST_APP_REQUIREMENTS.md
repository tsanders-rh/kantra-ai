# Test Application Requirements for Case Study

## Goal
Create a realistic Java EE 8 application that demonstrates typical javax → jakarta migration challenges and showcases kantra-ai's capabilities.

## Application Scope: "Enterprise Task Manager"

A realistic CRUD application with common Java EE patterns.

### Features
1. User authentication/authorization
2. CRUD operations for tasks
3. REST API endpoints
4. JPA persistence
5. Servlet filters and listeners
6. Bean validation
7. JSON serialization
8. Basic security

### Technology Stack
- Java 11
- Maven 3.x
- Java EE 8 (javax.* packages)
- JPA 2.2 (javax.persistence.*)
- JAX-RS 2.1 (javax.ws.rs.*)
- Servlet 4.0 (javax.servlet.*)
- Bean Validation 2.0 (javax.validation.*)
- JSON-B (javax.json.*)
- JUnit 5 for testing
- H2 database (in-memory for tests)

## Violation Types to Include

### 1. Simple Package Imports (Trivial)
```java
// ~50-100 occurrences across files
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
```

### 2. JPA Entity Annotations (Low)
```java
// ~20-30 entities
import javax.persistence.*;

@Entity
@Table(name = "tasks")
public class Task {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false)
    private String title;

    @ManyToOne
    @JoinColumn(name = "user_id")
    private User assignedTo;
}
```

### 3. JAX-RS REST Endpoints (Low-Medium)
```java
// ~10-15 REST controllers
import javax.ws.rs.*;
import javax.ws.rs.core.*;

@Path("/api/tasks")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class TaskResource {
    @GET
    @Path("/{id}")
    public Response getTask(@PathParam("id") Long id) {
        // ...
    }
}
```

### 4. Servlet Filters (Medium)
```java
import javax.servlet.*;
import javax.servlet.annotation.WebFilter;

@WebFilter("/*")
public class AuthenticationFilter implements Filter {
    @Override
    public void doFilter(ServletRequest request,
                        ServletResponse response,
                        FilterChain chain) {
        // ...
    }
}
```

### 5. Bean Validation (Low)
```java
import javax.validation.constraints.*;

public class TaskDTO {
    @NotNull
    @Size(min = 1, max = 200)
    private String title;

    @Email
    private String assigneeEmail;
}
```

### 6. JSON Processing (Low)
```java
import javax.json.bind.Jsonb;
import javax.json.bind.JsonbBuilder;
```

### 7. Configuration Files (Medium)

**persistence.xml:**
```xml
<persistence xmlns="http://xmlns.jcp.org/xml/ns/persistence"
             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
             xsi:schemaLocation="http://xmlns.jcp.org/xml/ns/persistence
             http://xmlns.jcp.org/xml/ns/persistence/persistence_2_2.xsd"
             version="2.2">
```

**web.xml:**
```xml
<web-app xmlns="http://xmlns.jcp.org/xml/ns/javaee"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://xmlns.jcp.org/xml/ns/javaee
         http://xmlns.jcp.org/xml/ns/javaee/web-app_4_0.xsd"
         version="4.0">
```

### 8. Dependency Injection (Low-Medium)
```java
import javax.inject.Inject;
import javax.inject.Named;
import javax.enterprise.context.RequestScoped;

@Named
@RequestScoped
public class TaskService {
    @Inject
    private TaskRepository repository;
}
```

## Expected Violation Counts

Target: **200-300 violations** total

| Category | Violations | Incidents | Complexity |
|----------|-----------|-----------|------------|
| Servlet imports | 15-20 | 40-60 | Trivial |
| JPA imports | 10-15 | 50-80 | Trivial |
| JAX-RS imports | 8-12 | 30-50 | Low |
| Validation imports | 5-8 | 20-30 | Low |
| JSON imports | 3-5 | 10-20 | Low |
| CDI imports | 5-8 | 15-25 | Low |
| XML namespaces | 3-5 | 3-5 | Medium |
| **Total** | **~60** | **~230** | Mixed |

## Test Coverage Requirements

- **Unit tests:** 80%+ coverage
- **Integration tests:** All REST endpoints
- **Persistence tests:** All JPA operations
- **Functional tests:** Key user workflows

**Total tests target:** 50-80 tests

## Project Structure

```
enterprise-task-manager/
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/example/taskmanager/
│   │   │       ├── model/           # JPA entities
│   │   │       │   ├── Task.java
│   │   │       │   ├── User.java
│   │   │       │   └── Category.java
│   │   │       ├── repository/      # Data access
│   │   │       │   ├── TaskRepository.java
│   │   │       │   └── UserRepository.java
│   │   │       ├── service/         # Business logic
│   │   │       │   ├── TaskService.java
│   │   │       │   └── UserService.java
│   │   │       ├── rest/            # REST endpoints
│   │   │       │   ├── TaskResource.java
│   │   │       │   ├── UserResource.java
│   │   │       │   └── AuthResource.java
│   │   │       ├── dto/             # Data transfer objects
│   │   │       │   ├── TaskDTO.java
│   │   │       │   └── UserDTO.java
│   │   │       ├── filter/          # Servlet filters
│   │   │       │   ├── AuthFilter.java
│   │   │       │   └── CorsFilter.java
│   │   │       └── exception/       # Exception handling
│   │   │           └── TaskNotFoundException.java
│   │   └── resources/
│   │       ├── META-INF/
│   │       │   └── persistence.xml
│   │       └── application.properties
│   └── test/
│       └── java/
│           └── com/example/taskmanager/
│               ├── repository/
│               ├── service/
│               └── rest/
├── webapp/
│   └── WEB-INF/
│       └── web.xml
├── pom.xml
└── README.md
```

## Maven Dependencies (Java EE 8)

```xml
<dependencies>
    <!-- Java EE 8 API -->
    <dependency>
        <groupId>javax</groupId>
        <artifactId>javaee-api</artifactId>
        <version>8.0</version>
        <scope>provided</scope>
    </dependency>

    <!-- Or individual specs -->
    <dependency>
        <groupId>javax.servlet</groupId>
        <artifactId>javax.servlet-api</artifactId>
        <version>4.0.1</version>
    </dependency>

    <dependency>
        <groupId>javax.persistence</groupId>
        <artifactId>javax.persistence-api</artifactId>
        <version>2.2</version>
    </dependency>

    <dependency>
        <groupId>javax.ws.rs</groupId>
        <artifactId>javax.ws.rs-api</artifactId>
        <version>2.1.1</version>
    </dependency>

    <!-- Testing -->
    <dependency>
        <groupId>org.junit.jupiter</groupId>
        <artifactId>junit-jupiter</artifactId>
        <version>5.9.0</version>
        <scope>test</scope>
    </dependency>

    <dependency>
        <groupId>com.h2database</groupId>
        <artifactId>h2</artifactId>
        <version>2.1.214</version>
        <scope>test</scope>
    </dependency>
</dependencies>
```

## Success Criteria

After migration with kantra-ai:

✅ All tests pass (50-80 tests, 100% passing)
✅ Clean build with Maven
✅ No compilation errors
✅ 90%+ success rate on violations
✅ All high-confidence fixes correct
✅ Low-confidence fixes properly flagged
✅ Total cost < $5
✅ Execution time < 15 minutes

## Time to Build

**Estimated effort:** 4-6 hours

- Project setup: 30 min
- Model layer (3 entities): 45 min
- Repository layer: 30 min
- Service layer: 45 min
- REST endpoints: 1 hour
- Filters/config: 30 min
- Unit tests: 1 hour
- Integration tests: 1 hour

**Or use AI to generate:** 30 minutes with Claude/GPT-4 to scaffold

## Benefits of Custom Test App

1. **Control:** You know exactly what's in it
2. **Validation:** You know the "right answer"
3. **Reproducibility:** Others can clone and verify
4. **Documentation:** Can document every decision
5. **Marketing:** "We built a realistic app to test it"
6. **Benchmark:** Can become a standard test case
7. **Showcase:** Can highlight specific kantra-ai features

## Alternative: Use AI to Generate It

Since you're showcasing AI capabilities, why not use AI to generate the test app?

```bash
# Use Claude/GPT-4 to generate the entire test app
# Then use kantra-ai to migrate it
# Shows: "AI generated → AI migrated"
```

This could be a compelling narrative: "AI-generated app, AI-migrated. Zero manual work."
