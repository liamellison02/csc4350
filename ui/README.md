# helmsman ui

react + vite + typescript spa for the helmsman control plane. it wires
the auth flow end to end: login gets a jwt, the token lives in an auth
context and localStorage, and a protected route gates the fleet dashboard.

## stack

- react 19 + vite 8 + typescript
- react-router-dom for routing
- vitest + testing-library for tests

## layout

| path                            | role                                       |
| ------------------------------- | ------------------------------------------ |
| src/lib/api.ts                  | typed client for the management api        |
| src/auth/context.ts             | auth context + token storage key           |
| src/auth/AuthContext.tsx        | AuthProvider: token/user state, rehydrate  |
| src/auth/useAuth.ts             | useAuth hook                               |
| src/components/ProtectedRoute.tsx | redirects to /login when unauthenticated |
| src/pages/Login.tsx             | credentials form, error banner             |
| src/pages/Dashboard.tsx         | fleet table with status badges             |

## config

copy `.env.example` to `.env` and set `VITE_API_URL` if the api is not on
`http://localhost:8000`.

## develop

```
npm install
npm run dev      # vite dev server on http://localhost:5173
```

## test and build

```
npm test         # vitest run
npm run build    # tsc -b && vite build
npm run lint     # eslint
```

## demo credentials

the seeded api accepts:

- `admin@helmsman.local` / `admin123!`
- `operator@helmsman.local` / `operator123!`
- `viewer@helmsman.local` / `viewer123!`
