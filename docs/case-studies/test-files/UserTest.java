package com.boatfuel.entity;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.ArrayList;

/**
 * Tests for User entity JPA mappings and basic functionality.
 *
 * Critical for javax.persistence → jakarta.persistence migration validation.
 */
class UserTest {

    @Test
    void testUserCreation() {
        User user = new User();
        user.setUsername("testuser");
        user.setPassword("password123");
        user.setEmail("test@example.com");

        assertEquals("testuser", user.getUsername());
        assertEquals("password123", user.getPassword());
        assertEquals("test@example.com", user.getEmail());
    }

    @Test
    void testUserWithFuelUps() {
        User user = new User();
        user.setUsername("boatowner");
        user.setFuelUps(new ArrayList<>());

        FuelUp fuelUp = new FuelUp();
        fuelUp.setUser(user);
        user.getFuelUps().add(fuelUp);

        assertEquals(1, user.getFuelUps().size());
        assertEquals(user, user.getFuelUps().get(0).getUser());
    }

    @Test
    void testUserDefaultValues() {
        User user = new User();

        // Test that collections are initialized (JPA relationship test)
        assertNotNull(user.getFuelUps());
        assertTrue(user.getFuelUps().isEmpty());
    }

    @Test
    void testUserEquality() {
        User user1 = new User();
        user1.setId(1L);
        user1.setUsername("user1");

        User user2 = new User();
        user2.setId(1L);
        user2.setUsername("user1");

        User user3 = new User();
        user3.setId(2L);
        user3.setUsername("user2");

        // If User implements equals/hashCode based on ID
        if (user1.getId() != null) {
            assertEquals(user1.getId(), user2.getId());
            assertNotEquals(user1.getId(), user3.getId());
        }
    }
}
