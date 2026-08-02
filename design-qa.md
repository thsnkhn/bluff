# TUI design QA

Final result: passed

## Visual sources

- Launch structure and Charm palette: `/Users/tahseen/.codex/generated_images/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/exec-ee3a23b7-d718-450e-a554-fa15aa641670.png`
- Logged-in heading treatment: `/Users/tahseen/Documents/Screenshots/Screenshot 2026-08-02 at 5.23.40 PM.png`
- Four-suit reference: `/Users/tahseen/Downloads/casino-card-suits-icon-set-vector-44265469.jpg.avif`

## Implementation captures

- Launch menu: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/home-fixed-gradient-logo.png`
- Centered login: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/login.png`
- Dashboard: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/dashboard.png`

## Checks

- Natural terminal background is preserved; no full-screen color fill remains.
- The launch composition is centered, menu descriptions are removed, and focus remains visually clear.
- The login flow is centered and uses Huh's native Charm theme.
- The supplied filled `BLUFF` artwork keeps its original geometry with fixed-width gradient rows, clear tagline spacing, and a compact fallback for narrow terminals.
- Dashboard headings use the requested violet title and slash rule.
- Loading operations use the Bubbles spinner.
- Keyboard and mouse share the same launch-menu focus; dashboard actions are clickable.
- Layouts were checked at 110 by 34 cells with no cropped or overlapping content.
