package com.example.config;

import java.util.Properties;

public class DatabaseConfig {

    public static Properties getConnectionProperties() {
        Properties props = new Properties();

        // Hardcoded database credentials - should use environment variables
        props.setProperty("db.host", "localhost");
        props.setProperty("db.port", "3306");
        props.setProperty("db.name", "mydb");
        props.setProperty("db.user", "root");
        props.setProperty("db.password", "SuperSecret123");

        return props;
    }

    public static String getAdminPassword() {
        // Another hardcoded credential
        return "admin123";
    }
}
