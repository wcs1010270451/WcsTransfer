# Frontend

Frontend admin console for WcsTransfer.

## Stack

- React
- Vite
- Ant Design
- React Router
- Zustand

## Run

```powershell
npm install
npm run dev
```

The local Vite dev server runs on `http://localhost:3211`.

## Environment

Create `.env` from `.env.example` if you want to override defaults:

- `VITE_API_BASE_URL=http://localhost:8080`
- `VITE_APP_BASE_PATH=/console/`

The admin token is entered from the top-right settings drawer after the app starts.
It is stored in `sessionStorage` only for the current browser session.
