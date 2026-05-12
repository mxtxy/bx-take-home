"use client";

import Alert from "@mui/joy/Alert";
import Box from "@mui/joy/Box";
import Button from "@mui/joy/Button";
import FormControl from "@mui/joy/FormControl";
import FormLabel from "@mui/joy/FormLabel";
import Input from "@mui/joy/Input";
import Sheet from "@mui/joy/Sheet";
import Stack from "@mui/joy/Stack";
import Typography from "@mui/joy/Typography";
import { FormEvent, useState } from "react";
import { login } from "../../lib/api";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await doLogin(email, password);
  }

  async function doLogin(nextEmail: string, nextPassword: string) {
    setError("");
    try {
      const { user } = await login(nextEmail, nextPassword);
      window.location.href = user.role === "manager" ? "/manager/quotes" : "/technician";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed.");
    }
  }

  return (
    <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center", bgcolor: "background.body", px: 2 }}>
      <Sheet variant="outlined" sx={{ width: "100%", maxWidth: 420, p: 3 }}>
        <Stack spacing={2.5} component="form" onSubmit={submit}>
          <Box>
            <Typography component="h1" level="h1" sx={{ fontSize: "1.8rem" }}>
              Brix Scheduler
            </Typography>
            <Typography color="neutral" level="body-sm">
              Sign in with a seeded account.
            </Typography>
          </Box>
          {error ? <Alert color="danger">{error}</Alert> : null}
          <FormControl>
            <FormLabel htmlFor="email">Email</FormLabel>
            <Input
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="email"
              slotProps={{ input: { id: "email" } }}
            />
          </FormControl>
          <FormControl>
            <FormLabel htmlFor="password">Password</FormLabel>
            <Input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              slotProps={{ input: { id: "password" } }}
            />
          </FormControl>
          <Button type="submit">
            Login
          </Button>
          <Stack direction={{ xs: "column", sm: "row" }} spacing={1}>
            <Button type="button" color="neutral" variant="outlined" onClick={() => doLogin("manager1@brix.test", "password123")} sx={{ flex: 1 }}>
              Manager demo
            </Button>
            <Button type="button" color="neutral" variant="outlined" onClick={() => doLogin("technician1@brix.test", "password123")} sx={{ flex: 1 }}>
              Technician demo
            </Button>
          </Stack>
        </Stack>
      </Sheet>
    </Box>
  );
}
