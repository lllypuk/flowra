# Dark Mode Toggle

**Priority:** 3 (Quality of Life)
**Status:** Complete

## Context

CSS already supports `prefers-color-scheme` via Pico CSS, and high-contrast mode CSS variables exist. However, there is no manual toggle in the UI — the app only follows OS preference.

## Deliverables

- [x] Add dark/light mode toggle button in navbar (sun/moon icon)
- [x] Store preference in localStorage
- [x] Apply `data-theme="dark"` or `data-theme="light"` on `<html>` element (Pico CSS convention)
- [x] Default to OS preference if no stored choice
- [x] Smooth transition on toggle (CSS transition on background-color, color)
- [x] Persist across page navigations (read from localStorage on page load)
- [x] Update toggle icon to reflect current state

## Technical Notes

- Pico CSS v2 supports `data-theme` attribute natively
- Add toggle logic to app.js (small addition)
- SVG sun/moon icons inline or from existing icon set
- No backend changes needed — purely client-side
