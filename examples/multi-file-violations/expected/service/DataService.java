package com.example.service;

import jakarta.servlet.ServletContext;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;

public class DataService {

    private ServletContext context;

    public DataService(ServletContext context) {
        this.context = context;
    }

    public Connection getDatabaseConnection() throws SQLException {
        // Credentials loaded from environment variables
        String url = System.getenv("DB_URL");
        String username = System.getenv("DB_USERNAME");
        String password = System.getenv("DB_PASSWORD");

        if (url == null || username == null || password == null) {
            throw new IllegalStateException("Database credentials not configured. " +
                "Set DB_URL, DB_USERNAME, and DB_PASSWORD environment variables.");
        }

        return DriverManager.getConnection(url, username, password);
    }

    public String getConfigValue(String key) {
        return context.getInitParameter(key);
    }
}
