"use client";

import { Box, CircularProgress } from "@mui/material";
import { useEffect } from "react";
import { getMe } from "../lib/api";

export default function HomePage() {
  useEffect(() => {
    getMe()
      .then(({ user }) => {
        window.location.href = user.role === "manager" ? "/manager" : "/technician";
      })
      .catch(() => {
        window.location.href = "/login";
      });
  }, []);

  return (
    <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center" }}>
      <CircularProgress aria-label="Loading" />
    </Box>
  );
}
