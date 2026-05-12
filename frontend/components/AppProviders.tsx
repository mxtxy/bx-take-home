"use client";

import { CssBaseline, ThemeProvider, createTheme } from "@mui/material";
import type { ReactNode } from "react";

const theme = createTheme({
  palette: {
    mode: "light",
    primary: { main: "#24536b" },
    secondary: { main: "#6b5b2a" },
    success: { main: "#2f6d4f" },
    warning: { main: "#9a5a18" },
    background: { default: "#f7f8f5" }
  },
  shape: {
    borderRadius: 6
  },
  typography: {
    fontFamily: "Arial, Helvetica, sans-serif"
  }
});

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  );
}
