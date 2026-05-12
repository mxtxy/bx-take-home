"use client";

import { Alert, Box, Button, Container, Stack, TextField, Typography } from "@mui/material";
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
      window.location.href = user.role === "manager" ? "/manager" : "/technician";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed.");
    }
  }

  return (
    <Container maxWidth="xs" sx={{ py: 8 }}>
      <Stack spacing={3} component="form" onSubmit={submit}>
        <Box>
          <Typography component="h1" variant="h4">
            Brix Scheduler
          </Typography>
          <Typography color="text.secondary">Sign in with a seeded account.</Typography>
        </Box>
        {error ? <Alert severity="error">{error}</Alert> : null}
        <TextField label="Email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" />
        <TextField
          label="Password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
        />
        <Button type="submit" variant="contained">
          Login
        </Button>
        <Stack direction="row" spacing={1}>
          <Button
            type="button"
            variant="outlined"
            onClick={() => doLogin("manager1@brix.test", "password123")}
          >
            Manager demo
          </Button>
          <Button
            type="button"
            variant="outlined"
            onClick={() => doLogin("technician1@brix.test", "password123")}
          >
            Technician demo
          </Button>
        </Stack>
      </Stack>
    </Container>
  );
}
