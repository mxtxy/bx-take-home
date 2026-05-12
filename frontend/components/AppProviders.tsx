"use client";

import CssBaseline from "@mui/joy/CssBaseline";
import GlobalStyles from "@mui/joy/GlobalStyles";
import { CssVarsProvider } from "@mui/joy/styles";
import type { ReactNode } from "react";

const uiCornerRadius = "12px";

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <CssVarsProvider defaultMode="system" modeStorageKey="brix-color-scheme">
      <GlobalStyles
        styles={{
          ":root": {
            "--brix-ui-corner-radius": uiCornerRadius,
            "--TableCell-cornerRadius": "var(--brix-ui-corner-radius)",
            "--unstable_actionRadius": "var(--brix-ui-corner-radius)"
          },
          ".MuiSheet-root": {
            borderRadius: "var(--brix-ui-corner-radius)"
          },
          ".MuiTable-root": {
            "--TableCell-cornerRadius": "var(--brix-ui-corner-radius)"
          }
        }}
      />
      <CssBaseline />
      {children}
    </CssVarsProvider>
  );
}
