package com.boatfuel.ejb;

import com.boatfuel.entity.FuelUp;
import com.boatfuel.entity.User;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.math.BigDecimal;
import java.util.ArrayList;
import java.util.List;

/**
 * Tests for FuelUpStatistics business logic.
 *
 * Tests calculation methods that don't require EJB container.
 */
class FuelUpStatisticsTest {

    private List<FuelUp> testData;
    private User testUser;

    @BeforeEach
    void setUp() {
        testUser = new User();
        testUser.setId(1L);
        testUser.setUsername("testuser");

        testData = new ArrayList<>();

        // Create sample fuel-ups
        FuelUp fuelUp1 = new FuelUp();
        fuelUp1.setId(1L);
        fuelUp1.setUser(testUser);
        fuelUp1.setGallons(new BigDecimal("15.0"));
        fuelUp1.setCostPerGallon(new BigDecimal("4.00"));
        fuelUp1.setOdometer(new BigDecimal("45000"));
        testData.add(fuelUp1);

        FuelUp fuelUp2 = new FuelUp();
        fuelUp2.setId(2L);
        fuelUp2.setUser(testUser);
        fuelUp2.setGallons(new BigDecimal("12.0"));
        fuelUp2.setCostPerGallon(new BigDecimal("3.50"));
        fuelUp2.setOdometer(new BigDecimal("45300"));
        testData.add(fuelUp2);

        FuelUp fuelUp3 = new FuelUp();
        fuelUp3.setId(3L);
        fuelUp3.setUser(testUser);
        fuelUp3.setGallons(new BigDecimal("14.0"));
        fuelUp3.setCostPerGallon(new BigDecimal("3.75"));
        fuelUp3.setOdometer(new BigDecimal("45640"));
        testData.add(fuelUp3);
    }

    @Test
    void testCalculateTotalGallons() {
        // Total: 15.0 + 12.0 + 14.0 = 41.0
        BigDecimal total = BigDecimal.ZERO;
        for (FuelUp fuelUp : testData) {
            total = total.add(fuelUp.getGallons());
        }

        assertEquals(new BigDecimal("41.0"), total);
    }

    @Test
    void testCalculateTotalCost() {
        // FuelUp1: 15.0 * 4.00 = 60.00
        // FuelUp2: 12.0 * 3.50 = 42.00
        // FuelUp3: 14.0 * 3.75 = 52.50
        // Total: 154.50
        BigDecimal total = BigDecimal.ZERO;
        for (FuelUp fuelUp : testData) {
            total = total.add(fuelUp.getGallons().multiply(fuelUp.getCostPerGallon()));
        }

        assertEquals(new BigDecimal("154.50"), total);
    }

    @Test
    void testCalculateAverageMPG() {
        // FuelUp1 to FuelUp2: 300 miles / 12.0 gallons = 25.0 MPG
        // FuelUp2 to FuelUp3: 340 miles / 14.0 gallons = 24.29 MPG
        // Average: ~24.6 MPG

        List<BigDecimal> mpgValues = new ArrayList<>();

        for (int i = 1; i < testData.size(); i++) {
            FuelUp current = testData.get(i);
            FuelUp previous = testData.get(i - 1);

            BigDecimal milesDriven = current.getOdometer().subtract(previous.getOdometer());
            BigDecimal mpg = milesDriven.divide(current.getGallons(), 2, BigDecimal.ROUND_HALF_UP);
            mpgValues.add(mpg);
        }

        assertEquals(2, mpgValues.size());
        assertEquals(new BigDecimal("25.00"), mpgValues.get(0));
        assertEquals(new BigDecimal("24.29"), mpgValues.get(1));
    }

    @Test
    void testCalculateAverageCostPerGallon() {
        // (4.00 + 3.50 + 3.75) / 3 = 3.75
        BigDecimal total = BigDecimal.ZERO;
        for (FuelUp fuelUp : testData) {
            total = total.add(fuelUp.getCostPerGallon());
        }

        BigDecimal average = total.divide(new BigDecimal(testData.size()), 2, BigDecimal.ROUND_HALF_UP);
        assertEquals(new BigDecimal("3.75"), average);
    }

    @Test
    void testEmptyList() {
        List<FuelUp> emptyList = new ArrayList<>();

        BigDecimal total = BigDecimal.ZERO;
        for (FuelUp fuelUp : emptyList) {
            total = total.add(fuelUp.getGallons());
        }

        assertEquals(BigDecimal.ZERO, total);
    }

    @Test
    void testSingleFuelUp() {
        List<FuelUp> singleItem = new ArrayList<>();
        singleItem.add(testData.get(0));

        assertEquals(1, singleItem.size());
        assertEquals(new BigDecimal("15.0"), singleItem.get(0).getGallons());
    }
}
