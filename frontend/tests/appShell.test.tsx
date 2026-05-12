import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import RootLayout, { metadata } from "../app/layout";
import { AppProviders } from "../components/AppProviders";

describe("app shell", () => {
  test("metadata describes the app", () => {
    expect(metadata).toEqual({
      title: "Brix Scheduler",
      description: "Service scheduling and notification system"
    });
  });

  test("AppProviders renders children inside the MUI theme", () => {
    render(
      <AppProviders>
        <span>Provided child</span>
      </AppProviders>
    );

    expect(screen.getByText("Provided child")).toBeInTheDocument();
  });

  test("RootLayout wraps children with html and body elements", () => {
    const element = RootLayout({ children: <span>Layout child</span> });

    expect(element.type).toBe("html");
    expect(element.props.lang).toBe("en");
    expect(element.props.children.type).toBe("body");
    expect(element.props.children.props.children.type).toBe(AppProviders);
  });
});
