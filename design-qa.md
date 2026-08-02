# TUI design QA

final result: passed

## Visual sources

- Launch structure and Charm palette: `/Users/tahseen/.codex/generated_images/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/exec-ee3a23b7-d718-450e-a554-fa15aa641670.png`
- Logged-in heading treatment: `/Users/tahseen/Documents/Screenshots/Screenshot 2026-08-02 at 5.23.40 PM.png`
- Four-suit reference: `/Users/tahseen/Downloads/casino-card-suits-icon-set-vector-44265469.jpg.avif`
- Selected Users structure (adapted from Players option 2): `/Users/tahseen/.codex/generated_images/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/exec-1f0996fc-b315-4daa-bd52-e5695ee5af51.png`

## Implementation captures

- Launch menu: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/home-fixed-gradient-logo.png`
- Centered login: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/login.png`
- Dashboard: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/dashboard.png`
- Users directory: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/users.png`
- Side-by-side Users comparison: `/Users/tahseen/.codex/visualizations/2026/08/02/019fc1d5-a5b3-7d82-80da-fdfcef6823ec/bluff-tui/users-comparison.png`

## Checks

- Natural terminal background is preserved; no full-screen color fill remains.
- The launch composition is centered, menu descriptions are removed, and focus remains visually clear.
- The login flow is centered and uses Huh's native Charm theme.
- The supplied filled `BLUFF` artwork keeps its original geometry with fixed-width gradient rows, clear tagline spacing, and a compact fallback for narrow terminals.
- Dashboard headings use the requested violet title and slash rule.
- Loading operations use the Bubbles spinner.
- Keyboard and mouse share the same launch-menu focus; dashboard actions are clickable.
- Layouts were checked at 110 by 34 cells with no cropped or overlapping content.
- The authenticated menu reuses the centered launch composition, shows `@username` and role, and exposes Users only to admins.
- The Users directory preserves the selected centered heading, identity, connection state, top shortcut strip, focused list, and bottom-pinned help bar.
- Player-only standings were intentionally replaced with account username and role data; `HOST` becomes the authenticated `ADMIN` role.
- The create-user flow reuses the centered heading and Charm-themed Huh form, including loading, error, and success states.
- Mouse hit regions cover authenticated menu items, user rows, and every Users shortcut.

## QA history

- Pass 1: the shortcut strip used a filled background and visually split at trailing terminal spaces (P2).
- Pass 2: replaced it with a Lip Gloss bordered shortcut strip and added the user-count summary; comparison passed with only intentional account-domain differences remaining.
- Pass 3: removed the repeated tagline, replaced the menu prompt with `Choose your next move.`, and moved connection state into one reusable bottom help bar used by every screen.
