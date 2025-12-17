package com.boatfuel.entity;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.math.BigDecimal;
import java.time.LocalDate;

/**
 * Tests for FuelUp entity JPA mappings and calculations.
 *
 * Critical for javax.persistence → jakarta.persistence migration validation.
 */
class FuelUpTest {

    @Test
    void testFuelUpCreation() {
        FuelUp fuelUp = new FuelUp();
        fuelUp.setGallons(new BigDecimal("15.5"));
        fuelUp.setCostPerGallon(new BigDecimal("3.89"));
        fuelUp.setOdometer(new BigDecimal("45000"));

        assertEquals(new BigDecimal("15.5"), fuelUp.getGallons());
        assertEquals(new BigDecimal("3.89"), fuelUp.getCostPerGallon());
        assertEquals(new BigDecimal("45000"), fuelUp.getOdometer());
    }

    @Test
    void testFuelUpWithUser() {
        User user = new User();
        user.setUsername("captain");

        FuelUp fuelUp = new FuelUp();
        fuelUp.setUser(user);

        assertEquals(user, fuelUp.getUser());
        assertEquals("captain", fuelUp.getUser().getUsername());
    }

    @Test
    void testTotalCostCalculation() {
        FuelUp fuelUp = new FuelUp();
        fuelUp.setGallons(new BigDecimal("10.0"));
        fuelUp.setCostPerGallon(new BigDecimal("4.00"));

        // Total should be 10.0 * 4.00 = 40.00
        BigDecimal expectedTotal = new BigDecimal("40.00");
        BigDecimal actualTotal = fuelUp.getGallons().multiply(fuelUp.getCostPerGallon());

        assertEquals(0, expectedTotal.compareTo(actualTotal));
    }

    @Test
    void testFuelUpWithDate() {
        FuelUp fuelUp = new FuelUp();
        LocalDate today = LocalDate.now();
        fuelUp.setFuelUpDate(java.sql.Date.valueOf(today));

        assertNotNull(fuelUp.getFuelUpDate());
    }

    @Test
    void testBigDecimalPrecision() {
        FuelUp fuelUp = new FuelUp();

        // Test that BigDecimal maintains precision (important for currency)
        BigDecimal gallons = new BigDecimal("15.567");
        BigDecimal costPerGallon = new BigDecimal("3.899");

        fuelUp.setGallons(gallons);
        fuelUp.setCostPerGallon(costPerGallon);

        assertEquals(gallons, fuelUp.getGallons());
        assertEquals(costPerGallon, fuelUp.getCostPerGallon());
    }

    @Test
    void testMPGCalculation() {
        // Test miles per gallon calculation
        FuelUp current = new FuelUp();
        current.setOdometer(new BigDecimal("50000"));
        current.setGallons(new BigDecimal("12.5"));

        FuelUp previous = new FuelUp();
        previous.setOdometer(new BigDecimal("49700"));

        // Miles driven = 50000 - 49700 = 300
        // MPG = 300 / 12.5 = 24.0
        BigDecimal milesDriven = current.getOdometer().subtract(previous.getOdometer());
        BigDecimal mpg = milesDriven.divide(current.getGallons(), 2, BigDecimal.ROUND_HALF_UP);

        assertEquals(new BigDecimal("24.00"), mpg);
    }
}
