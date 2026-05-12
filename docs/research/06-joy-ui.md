# Joy UI Research

## Sources

- MUI Joy UI overview: https://mui.com/joy-ui/getting-started/
- Joy UI dark mode guide: https://mui.com/joy-ui/customization/dark-mode/
- Joy UI CSS variables guide: https://mui.com/joy-ui/customization/using-css-variables/
- Joy UI theme colors: https://mui.com/joy-ui/customization/theme-colors/
- Joy UI global variants: https://mui.com/joy-ui/main-features/global-variants/
- Joy UI shadows: https://mui.com/joy-ui/customization/theme-shadow/
- Joy UI Button component: https://mui.com/joy-ui/react-button/
- Joy UI Sheet component: https://mui.com/joy-ui/react-sheet/
- Joy UI Select/components list: https://mui.com/joy-ui/react-select/
- MUI X Date Pickers quickstart: https://mui.com/x/react-date-pickers/quickstart/

## Findings

Joy UI is MUI's non-Material design system. It provides the same kind of React component APIs as Material UI, but with Joy-specific primitives such as `Sheet`, `Stack`, `FormControl`, `Select`, `Option`, `Input`, `Table`, `Alert`, `Chip`, `IconButton`, and CSS-variable-first theming.

Joy UI is currently beta and MUI notes that development is on hold. This migration still uses it because the product requirement specifically asks for Joy UI, but the choice is documented for maintainability.

Dark mode should be implemented with Joy's `CssVarsProvider`, `useColorScheme`, and `InitColorSchemeScript` in the App Router root layout. This prevents color-scheme flicker and lets the UI support light, dark, and system-derived modes while keeping Joy's default light/dark palette mapping.

Joy's default semantic palettes already provide `primary`, `neutral`, `danger`, `success`, and `warning` colors with light and dark mappings. The frontend should not override these tokens or add a custom accent switcher unless the product explicitly needs brand customization.

Joy's global variants are the interaction hierarchy for the app: `solid` for the most important primary action, and `soft`, `outlined`, or `plain` for secondary and tertiary actions. Joy `Button` defaults to `solid`, so primary actions can use `<Button>` directly; secondary actions should opt into neutral `outlined` or `plain` variants.

Joy's default shadow scale is used by components such as Card and Menu. The frontend should avoid custom `boxShadow`, radius, and component-default overrides so Joy components keep their built-in depth and density.

Joy `Select` is a better fit than native-select styling for this app because it exposes an obvious combobox surface, placeholder text such as `Select a quote`, a default dropdown indicator, and accessible listbox options while keeping the visual language consistent with the rest of Joy UI.

MUI X Date Pickers were evaluated for the calendar interface, but the package has a Material UI peer dependency. To keep this migration Joy-first, the scheduler uses a custom Joy UI calendar/time-slot picker around the existing `datetime-local` value shape. This preserves the backend contract while giving managers an actual calendar workflow.
