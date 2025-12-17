# Test Plan: Boat Fuel Tracker J2EE

## Goal
Add comprehensive tests to validate that javax → jakarta migration doesn't break functionality.

## Test Coverage Target
- **Entity layer:** 100% (critical for JPA migration)
- **Service layer:** 80%+
- **Overall:** 70%+
- **Total tests:** 15-20 tests
- **Execution time:** <10 seconds

## Test Dependencies to Add

Add to `pom.xml`:

```xml
<!-- Test Dependencies -->
<dependency>
    <groupId>org.junit.jupiter</groupId>
    <artifactId>junit-jupiter</artifactId>
    <version>5.9.3</version>
    <scope>test</scope>
</dependency>

<dependency>
    <groupId>org.mockito</groupId>
    <artifactId>mockito-core</artifactId>
    <version>5.3.1</version>
    <scope>test</scope>
</dependency>

<dependency>
    <groupId>org.mockito</groupId>
    <artifactId>mockito-junit-jupiter</artifactId>
    <version>5.3.1</version>
    <scope>test</scope>
</dependency>

<!-- H2 for in-memory testing -->
<dependency>
    <groupId>com.h2database</groupId>
    <artifactId>h2</artifactId>
    <version>2.1.214</version>
    <scope>test</scope>
</dependency>

<!-- Hibernate for test persistence -->
<dependency>
    <groupId>org.hibernate</groupId>
    <artifactId>hibernate-core</artifactId>
    <version>5.6.15.Final</version>
    <scope>test</scope>
</dependency>
```

## Test Classes to Create

### 1. Entity Tests (JPA Critical)

**`src/test/java/com/boatfuel/entity/UserTest.java`**
- Test entity creation
- Test JPA annotations work
- Test persistence operations

**`src/test/java/com/boatfuel/entity/FuelUpTest.java`**
- Test entity creation
- Test relationships (User ↔ FuelUp)
- Test BigDecimal calculations

### 2. Service/Business Logic Tests

**`src/test/java/com/boatfuel/ejb/FuelUpStatisticsTest.java`**
- Test statistics calculations
- Test average MPG calculation
- Test total cost summation

### 3. Integration Tests

**`src/test/java/com/boatfuel/integration/FuelUpPersistenceTest.java`**
- Test full CRUD operations with H2
- Verify JPA mappings work end-to-end

## Test Configuration

**`src/test/resources/META-INF/persistence.xml`** (Test-specific)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<persistence xmlns="http://xmlns.jcp.org/xml/ns/persistence"
             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
             xsi:schemaLocation="http://xmlns.jcp.org/xml/ns/persistence
             http://xmlns.jcp.org/xml/ns/persistence/persistence_2_2.xsd"
             version="2.2">

    <persistence-unit name="test-pu" transaction-type="RESOURCE_LOCAL">
        <provider>org.hibernate.jpa.HibernatePersistenceProvider</provider>

        <class>com.boatfuel.entity.User</class>
        <class>com.boatfuel.entity.FuelUp</class>

        <properties>
            <!-- H2 in-memory database -->
            <property name="javax.persistence.jdbc.driver" value="org.h2.Driver"/>
            <property name="javax.persistence.jdbc.url" value="jdbc:h2:mem:testdb;DB_CLOSE_DELAY=-1"/>
            <property name="javax.persistence.jdbc.user" value="sa"/>
            <property name="javax.persistence.jdbc.password" value=""/>

            <!-- Hibernate settings -->
            <property name="hibernate.dialect" value="org.hibernate.dialect.H2Dialect"/>
            <property name="hibernate.hbm2ddl.auto" value="create-drop"/>
            <property name="hibernate.show_sql" value="false"/>
            <property name="hibernate.format_sql" value="true"/>
        </properties>
    </persistence-unit>
</persistence>
```

## Maven Surefire Plugin

Add to `pom.xml` build plugins:

```xml
<plugin>
    <groupId>org.apache.maven.plugins</groupId>
    <artifactId>maven-surefire-plugin</artifactId>
    <version>3.0.0</version>
    <configuration>
        <includes>
            <include>**/*Test.java</include>
        </includes>
    </configuration>
</plugin>
```

## Validation Commands

After adding tests:

```bash
# Run all tests
mvn clean test

# Run with coverage
mvn clean test jacoco:report

# Verify build (what kantra-ai will run)
mvn clean verify
```

## Expected Output

```
[INFO] Tests run: 18, Failures: 0, Errors: 0, Skipped: 0
[INFO] BUILD SUCCESS
```

## Migration Validation

These tests will catch:
- ✅ `javax.persistence.*` → `jakarta.persistence.*` issues
- ✅ `javax.ejb.*` → `jakarta.ejb.*` issues
- ✅ JPA annotation changes
- ✅ Entity manager factory issues
- ✅ Transaction handling changes
- ✅ Any breaking API changes

If tests pass after migration = **PROOF** the migration worked!
