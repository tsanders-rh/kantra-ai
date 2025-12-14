package com.example.service;

import javax.servlet.ServletContext;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;

public class DataService {

    private ServletContext context;

    public DataService(ServletContext context) {
        this.context = context;
    }

    public Connection getDatabaseConnection() throws SQLException {
        // Hardcoded credentials - security violation
        String url = "jdbc:mysql://localhost:3306/mydb";
        String username = "admin";
        String password = "Password123!";

        return DriverManager.getConnection(url, username, password);
    }

    public String getConfigValue(String key) {
        return context.getInitParameter(key);
    }
}
