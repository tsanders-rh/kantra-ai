package com.example.config;

import java.util.Properties;

public class DatabaseConfig {

    public static Properties getConnectionProperties() {
        Properties props = new Properties();

        // Load configuration from environment variables
        props.setProperty("db.host", System.getenv().getOrDefault("DB_HOST", "localhost"));
        props.setProperty("db.port", System.getenv().getOrDefault("DB_PORT", "3306"));
        props.setProperty("db.name", System.getenv().getOrDefault("DB_NAME", "mydb"));
        props.setProperty("db.user", System.getenv("DB_USER"));
        props.setProperty("db.password", System.getenv("DB_PASSWORD"));

        return props;
    }

    public static String getAdminPassword() {
        // Load admin password from environment variable
        String password = System.getenv("ADMIN_PASSWORD");
        if (password == null) {
            throw new IllegalStateException("ADMIN_PASSWORD environment variable not set");
        }
        return password;
    }
}
