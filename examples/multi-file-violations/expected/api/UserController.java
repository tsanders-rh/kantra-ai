package com.example.api;

import jakarta.servlet.http.HttpServlet;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;

public class UserController extends HttpServlet {

    @Override
    protected void doGet(HttpServletRequest req, HttpServletResponse resp) throws IOException {
        String userId = req.getParameter("id");

        // Fetch user from database
        resp.setContentType("application/json");
        resp.getWriter().write("{\"id\": \"" + userId + "\"}");
    }

    @Override
    protected void doPost(HttpServletRequest req, HttpServletResponse resp) throws IOException {
        // Create new user
        resp.setStatus(201);
        resp.getWriter().write("{\"status\": \"created\"}");
    }
}
