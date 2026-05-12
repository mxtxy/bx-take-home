"use client";

import Box from "@mui/joy/Box";
import CircularProgress from "@mui/joy/CircularProgress";
import { useEffect } from "react";
import { getMe } from "../lib/api";

export default function HomePage() {
  useEffect(() => {
    getMe()
      .then(({ user }) => {
        window.location.href = user.role === "manager" ? "/manager/quotes" : "/technician";
      })
      .catch(() => {
        window.location.href = "/login";
      });
  }, []);

  return (
    <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center", bgcolor: "background.body" }}>
      <CircularProgress aria-label="Loading" />
    </Box>
  );
}
