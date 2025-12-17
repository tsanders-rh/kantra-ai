# How to Add Tests to boat-fuel-tracker-j2ee

## Step 1: Add Test Dependencies to pom.xml

Add these dependencies in the `<dependencies>` section:

```xml
<!-- JUnit 5 -->
<dependency>
    <groupId>org.junit.jupiter</groupId>
    <artifactId>junit-jupiter</artifactId>
    <version>5.9.3</version>
    <scope>test</scope>
</dependency>

<!-- Mockito -->
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

<!-- H2 Database for testing -->
<dependency>
    <groupId>com.h2database</groupId>
    <artifactId>h2</artifactId>
    <version>2.1.214</version>
    <scope>test</scope>
</dependency>
```

## Step 2: Add Surefire Plugin

Add to `<build><plugins>`:

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

## Step 3: Create Test Directory Structure

```bash
cd boat-fuel-tracker-j2ee

mkdir -p src/test/java/com/boatfuel/entity
mkdir -p src/test/java/com/boatfuel/ejb
mkdir -p src/test/java/com/boatfuel/integration
mkdir -p src/test/resources/META-INF
```

## Step 4: Copy Test Files

Copy the test files to the appropriate locations:

```bash
# Entity tests
cp UserTest.java src/test/java/com/boatfuel/entity/
cp FuelUpTest.java src/test/java/com/boatfuel/entity/

# Business logic tests
cp FuelUpStatisticsTest.java src/test/java/com/boatfuel/ejb/

# Integration tests
cp FuelUpPersistenceTest.java src/test/java/com/boatfuel/integration/
```

## Step 5: Create Test persistence.xml

Create `src/test/resources/META-INF/persistence.xml`:

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

## Step 6: Run Tests

```bash
# Run all tests
mvn clean test

# Expected output:
# Tests run: 18-20, Failures: 0, Errors: 0, Skipped: 0
# BUILD SUCCESS
```

## Step 7: Commit Tests (Before Migration)

```bash
git add src/test pom.xml
git commit -m "Add comprehensive test suite for migration validation

- Add JUnit 5 tests for entity layer
- Add tests for business logic
- Add JPA integration tests with H2
- Tests validate javax.persistence API usage
- Will verify jakarta.persistence migration success"
git push
```

## What These Tests Validate

### Before Migration (javax.*)
- ✅ Tests use `javax.persistence.*` classes
- ✅ All tests pass (proves baseline works)
- ✅ Provides 70%+ code coverage

### After Migration (jakarta.*)
- ✅ kantra-ai will migrate `javax.persistence.*` → `jakarta.persistence.*`
- ✅ Tests will still pass if migration is correct
- ✅ Tests will fail if migration breaks anything

### Why This is Powerful

**Case Study Proof Points:**
- "All 18 tests passed before migration"
- "All 18 tests passed after migration"
- "kantra-ai successfully migrated javax → jakarta with 100% test pass rate"
- "Zero manual fixes required for test code"

## Verification Command for kantra-ai

After setting this up, use this command for your case study:

```bash
kantra-ai remediate \
  --analysis=./konveyor-analysis/output.yaml \
  --input=. \
  --provider=claude \
  --verify=at-end \
  --verify-build="mvn clean test" \
  --verify-fail-fast=true
```

This will:
1. Run the migration
2. Automatically run `mvn clean test`
3. Report if tests pass or fail
4. Give you concrete proof the migration worked!
