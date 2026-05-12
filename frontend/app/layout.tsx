import type { Metadata } from "next";
import type { ReactNode } from "react";
import { AppProviders } from "../components/AppProviders";

export const metadata: Metadata = {
  title: "Brix Scheduler",
  description: "Service scheduling and notification system"
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
