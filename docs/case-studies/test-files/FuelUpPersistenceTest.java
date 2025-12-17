package com.boatfuel.integration;

import com.boatfuel.entity.FuelUp;
import com.boatfuel.entity.User;
import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

import javax.persistence.EntityManager;
import javax.persistence.EntityManagerFactory;
import javax.persistence.Persistence;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

/**
 * Integration tests for JPA persistence operations.
 *
 * CRITICAL for javax.persistence → jakarta.persistence migration validation.
 * These tests use the actual JPA API and will break if migration fails.
 */
class FuelUpPersistenceTest {

    private static EntityManagerFactory emf;
    private EntityManager em;

    @BeforeAll
    static void setUpClass() {
        // Initialize EntityManagerFactory once for all tests
        // This uses javax.persistence classes that will be migrated
        emf = Persistence.createEntityManagerFactory("test-pu");
    }

    @AfterAll
    static void tearDownClass() {
        if (emf != null && emf.isOpen()) {
            emf.close();
        }
    }

    @BeforeEach
    void setUp() {
        em = emf.createEntityManager();
    }

    @AfterEach
    void tearDown() {
        if (em != null && em.isOpen()) {
            em.close();
        }
    }

    @Test
    void testPersistUser() {
        em.getTransaction().begin();

        User user = new User();
        user.setUsername("testuser");
        user.setPassword("password123");
        user.setEmail("test@example.com");

        em.persist(user);
        em.getTransaction().commit();

        assertNotNull(user.getId());
        assertTrue(user.getId() > 0);
    }

    @Test
    void testPersistFuelUp() {
        em.getTransaction().begin();

        // Create and persist user first
        User user = new User();
        user.setUsername("captain");
        user.setPassword("pass");
        user.setEmail("captain@boat.com");
        em.persist(user);

        // Create and persist fuel-up
        FuelUp fuelUp = new FuelUp();
        fuelUp.setUser(user);
        fuelUp.setGallons(new BigDecimal("15.5"));
        fuelUp.setCostPerGallon(new BigDecimal("3.89"));
        fuelUp.setOdometer(new BigDecimal("45000"));
        fuelUp.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now()));

        em.persist(fuelUp);
        em.getTransaction().commit();

        assertNotNull(fuelUp.getId());
        assertTrue(fuelUp.getId() > 0);
    }

    @Test
    void testFindUserById() {
        // Create user
        em.getTransaction().begin();
        User user = new User();
        user.setUsername("findme");
        user.setPassword("pass");
        user.setEmail("find@test.com");
        em.persist(user);
        em.getTransaction().commit();

        Long userId = user.getId();
        em.clear(); // Clear persistence context

        // Find user
        User found = em.find(User.class, userId);

        assertNotNull(found);
        assertEquals("findme", found.getUsername());
        assertEquals("find@test.com", found.getEmail());
    }

    @Test
    void testUserFuelUpRelationship() {
        em.getTransaction().begin();

        // Create user
        User user = new User();
        user.setUsername("boatowner");
        user.setPassword("pass");
        user.setEmail("owner@boat.com");
        em.persist(user);

        // Create multiple fuel-ups
        FuelUp fuelUp1 = new FuelUp();
        fuelUp1.setUser(user);
        fuelUp1.setGallons(new BigDecimal("10.0"));
        fuelUp1.setCostPerGallon(new BigDecimal("4.00"));
        fuelUp1.setOdometer(new BigDecimal("45000"));
        fuelUp1.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now()));
        em.persist(fuelUp1);

        FuelUp fuelUp2 = new FuelUp();
        fuelUp2.setUser(user);
        fuelUp2.setGallons(new BigDecimal("12.0"));
        fuelUp2.setCostPerGallon(new BigDecimal("3.50"));
        fuelUp2.setOdometer(new BigDecimal("45300"));
        fuelUp2.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now().minusDays(1)));
        em.persist(fuelUp2);

        em.getTransaction().commit();

        Long userId = user.getId();
        em.clear();

        // Query fuel-ups for user
        List<FuelUp> fuelUps = em.createQuery(
            "SELECT f FROM FuelUp f WHERE f.user.id = :userId ORDER BY f.fuelUpDate DESC",
            FuelUp.class)
            .setParameter("userId", userId)
            .getResultList();

        assertEquals(2, fuelUps.size());
        assertEquals(new BigDecimal("12.0"), fuelUps.get(0).getGallons()); // Most recent first
    }

    @Test
    void testUpdateFuelUp() {
        // Create fuel-up
        em.getTransaction().begin();
        User user = new User();
        user.setUsername("updater");
        user.setPassword("pass");
        user.setEmail("update@test.com");
        em.persist(user);

        FuelUp fuelUp = new FuelUp();
        fuelUp.setUser(user);
        fuelUp.setGallons(new BigDecimal("10.0"));
        fuelUp.setCostPerGallon(new BigDecimal("4.00"));
        fuelUp.setOdometer(new BigDecimal("45000"));
        fuelUp.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now()));
        em.persist(fuelUp);
        em.getTransaction().commit();

        Long fuelUpId = fuelUp.getId();
        em.clear();

        // Update fuel-up
        em.getTransaction().begin();
        FuelUp found = em.find(FuelUp.class, fuelUpId);
        found.setGallons(new BigDecimal("15.0"));
        em.getTransaction().commit();
        em.clear();

        // Verify update
        FuelUp updated = em.find(FuelUp.class, fuelUpId);
        assertEquals(new BigDecimal("15.0"), updated.getGallons());
    }

    @Test
    void testDeleteFuelUp() {
        // Create fuel-up
        em.getTransaction().begin();
        User user = new User();
        user.setUsername("deleter");
        user.setPassword("pass");
        user.setEmail("delete@test.com");
        em.persist(user);

        FuelUp fuelUp = new FuelUp();
        fuelUp.setUser(user);
        fuelUp.setGallons(new BigDecimal("10.0"));
        fuelUp.setCostPerGallon(new BigDecimal("4.00"));
        fuelUp.setOdometer(new BigDecimal("45000"));
        fuelUp.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now()));
        em.persist(fuelUp);
        em.getTransaction().commit();

        Long fuelUpId = fuelUp.getId();
        em.clear();

        // Delete fuel-up
        em.getTransaction().begin();
        FuelUp found = em.find(FuelUp.class, fuelUpId);
        em.remove(found);
        em.getTransaction().commit();
        em.clear();

        // Verify deletion
        FuelUp deleted = em.find(FuelUp.class, fuelUpId);
        assertNull(deleted);
    }

    @Test
    void testJPQLQuery() {
        // Create test data
        em.getTransaction().begin();
        User user = new User();
        user.setUsername("querier");
        user.setPassword("pass");
        user.setEmail("query@test.com");
        em.persist(user);

        for (int i = 0; i < 5; i++) {
            FuelUp fuelUp = new FuelUp();
            fuelUp.setUser(user);
            fuelUp.setGallons(new BigDecimal("10.0").add(new BigDecimal(i)));
            fuelUp.setCostPerGallon(new BigDecimal("4.00"));
            fuelUp.setOdometer(new BigDecimal("45000").add(new BigDecimal(i * 100)));
            fuelUp.setFuelUpDate(java.sql.Date.valueOf(LocalDate.now().minusDays(i)));
            em.persist(fuelUp);
        }
        em.getTransaction().commit();
        em.clear();

        // Query with JPQL
        List<FuelUp> results = em.createQuery(
            "SELECT f FROM FuelUp f WHERE f.user.username = :username ORDER BY f.fuelUpDate DESC",
            FuelUp.class)
            .setParameter("username", "querier")
            .getResultList();

        assertEquals(5, results.size());
        // Verify ordering (most recent first)
        assertTrue(results.get(0).getOdometer().compareTo(results.get(4).getOdometer()) < 0);
    }
}
